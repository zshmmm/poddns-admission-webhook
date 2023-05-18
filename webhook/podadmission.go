package webhook

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/go-logr/logr"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	opt "poddns-admission-webhook/cmd/options"
	"poddns-admission-webhook/pkg/tools"
)

type PodInject struct {
	DnsConfig DnsConfig `yaml:"dnsConfig"`
	// 将 Option 配置转换为 map，快速索引
	PodDNSCOnfigOptionsMap map[string]string
	// 将 Option 配置实例化为 []corev1.PodDNSConfigOption{}，启动时转换一次
	PodDNSConfigOptions []corev1.PodDNSConfigOption
	Log                logr.Logger
	Dcoder runtime.Decoder
}

func NewPodInject() (*PodInject, error) {
	// 初始化 Log
	pi := PodInject{
		DnsConfig:          DnsConfig{},
		PodDNSCOnfigOptionsMap:             map[string]string{},
		PodDNSConfigOptions: []corev1.PodDNSConfigOption{},
		Log:                logf.FromContext(context.TODO(), "logic", "podDnsConfigInject"),
		Dcoder: serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer(),
	}

	// 读取配置文件
	conf := filepath.Join(opt.Options.ConfDir, opt.Options.ConfFile)
	content, err := ioutil.ReadFile(conf)
	if err != nil {
		pi.Log.Error(err, "open config file failed", "file", conf)
		return nil, err
	}

	// 将配置转换为 golang 结构体
	// var PodInject PodInject
	if err := yaml.Unmarshal(content, &pi); err != nil {
		pi.Log.Error(err, "unmarshal config failed", "file", conf)
		return nil, err
	}

	// 初始化 PodInject.OptMap 和 PodInject.PodDNSConfigOption
	pi.optionsToPodDNSConfigOptions()

	return &pi, nil
}

// 将配置文件中的 Options 转换为 []corev1.PodDNSConfigOption{}
// 将配置文件中的 Options 存储到 PodInject.OptMap 方便检索
func (p *PodInject) optionsToPodDNSConfigOptions() {
	for _, v := range p.DnsConfig.Options {
		value := v.Value
		opt := corev1.PodDNSConfigOption{
			Name:  v.Name,
			Value: &value,
		}

		p.PodDNSConfigOptions = append(p.PodDNSConfigOptions, opt)
		p.PodDNSCOnfigOptionsMap[v.Name] = v.Value
	}
}

// 返回的 error 如果不为空，本次请求会被决绝 admission.Denied(err.Error())
// 在 sigs.k8s.io/controller-runtime@v0.14.6/pkg/webhook/admission/defaulter_custom.go 中
// 如下函数被 admission.Handle 调用
func (p *PodInject) Default(ctx context.Context, obj runtime.Object) error {
	// 参数 obj 为 api 请求的k8s资源对象，如想要获取原始请求 admission.Request 需要通过如下方式从 ctx 中获取
	// 在 admission.Handle 中会将原始请求放到 ctx 中，并提供了如下方法取出

	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		err := apierrors.NewInternalError(err)
		p.Log.Error(err, "admission.Request not found in context")
		// 这种情况不做处理直接返回
		return nil
	}

	// 如果是 tryRun 不做任何操作返回(https://kubernetes.io/zh-cn/docs/reference/access-authn-authz/extensible-admission-controllers/)
	if *req.DryRun {
		p.Log.Info("dry run inject ignore")
		return nil
	}

	pod, ok := obj.(*corev1.Pod)
	if !ok {
		// 外层函数会解析 apierrors.APIStatus 接口类型的错误，可以通过 apierrors 模块中的错误生成函数来初始化得到相应类型错误
		// 如果直接返回错误，则外层直接返回 admission.Denied(err.Error()) 错误
		err := apierrors.NewInternalError(fmt.Errorf("expected a pod but get %+v", obj))
		p.Log.Error(err, "")
		// 这种情况不做处理直接返回
		return nil
	}

	// 只注入 DNSPolicy 为 ClusterFirst 的 Pod
	if pod.Spec.DNSPolicy != corev1.DNSClusterFirst {
		p.Log.Info("inject ignore", "DNSPolicy", pod.Spec.DNSPolicy)
		return nil
	}

	// dnsConfig 注入
	p.dnsConfigInject(pod)
	p.Log.Info("pod inject spec.dnsConfig", "namespace", pod.GetNamespace(), "pod", pod.GetName(), "dnsConfig", pod.Spec.DNSConfig)

	return nil
}

func (p *PodInject) dnsConfigInject(pod *corev1.Pod) {
	// pod 不存在 dnsConfig 配置
	if pod.Spec.DNSConfig == nil {
		pod.Spec.DNSConfig = &corev1.PodDNSConfig{
			Nameservers: p.DnsConfig.NameServers,
			Searches:    p.DnsConfig.Searches,
			Options:     p.PodDNSConfigOptions,
		}
		return
	}

	// pod.Spec.DNSConfig.NameServers 注入
	p.dnsNameServersInject(pod)

	// pod.Spec.DNSConfig.Searches 注入
	p.dnsSearchesInject(pod)

	// pod.Spec.DNSConfig.Options 注入
	p.dnsOptionsInject(pod)

}

// DNSConfig.NameServers 注入函数
func (p *PodInject) dnsNameServersInject(pod *corev1.Pod) {
	dnsconf := pod.Spec.DNSConfig.DeepCopy()
	// 保证注入的 nameserver 在最前面，保障 localdns 生效
	ns := []string{}
	ns = append(ns, p.DnsConfig.NameServers...)

	for _, s := range dnsconf.Nameservers {
		if ok := tools.Contains(p.DnsConfig.NameServers, s); !ok {
			ns = append(ns, s)
		}
	}

	// TODO 如果 len(ns) >= 3 是否需要裁剪，还是全部保留

	pod.Spec.DNSConfig.Nameservers = ns
}

// DNSConfig.Searches 注入函数
func (p *PodInject) dnsSearchesInject(pod *corev1.Pod) {
	dnsconf := pod.Spec.DNSConfig.DeepCopy()
	// 注入的 search 在最后面
	sc := []string{}
	sc = append(sc, dnsconf.Searches...)
	for _, s := range p.DnsConfig.Searches {
		if ok := tools.Contains(dnsconf.Searches, s); !ok {
			sc = append(sc, s)
		}
	}
	pod.Spec.DNSConfig.Searches = sc
}

// DNSConfig.Options 注入函数
func (p *PodInject) dnsOptionsInject(pod *corev1.Pod) {
	// Options 无配置
	if len(pod.Spec.DNSConfig.Options) == 0 {
		pod.Spec.DNSConfig.Options = append(pod.Spec.DNSConfig.Options, p.PodDNSConfigOptions...)
		return
	}

	// Options 已有配置
	if len(pod.Spec.DNSConfig.Options) > 0 {
		newOption := []corev1.PodDNSConfigOption{}
		opts := make(map[string]string)
		// 一次遍历将 Options 转换为 map
		for _, v := range pod.Spec.DNSConfig.Options {
			opts[v.Name] = *v.Value
		}

		for name, value := range p.PodDNSCOnfigOptionsMap {
			if _, ok := opts[name]; !ok {
				v := value
				newOption = append(newOption, corev1.PodDNSConfigOption{
					Name:  name,
					Value: &v,
				})
			} else {
				// TODO 根据优先级配置是否覆盖原有配置
			}
		}
		if len(newOption) == 0 {
			return
		}
		pod.Spec.DNSConfig.Options = append(pod.Spec.DNSConfig.Options, newOption...)
	}
}