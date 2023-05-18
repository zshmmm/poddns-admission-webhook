package main

import (
	"flag"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	opt "poddns-admission-webhook/cmd/options"
	"poddns-admission-webhook/webhook"
)

var (
	scheme   = runtime.NewScheme()
	webhookLogger = log.Log.WithName("poddns-admission-webhook")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	log.SetLogger(zap.New())
}

func main() {

	flag.Parse()

	// 初始化 manager 实例
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		// webhook 服务端口
		Port: opt.Options.Port,
		// webhook 服务端证书目录，使用 controller-runtime 证书文件必须指定为：tls.key 和 tls.crt
		// webhook deployment 中使用的 secret 生成时必须满足当前需求
		CertDir:        opt.Options.CertDir,
		LeaderElection: opt.Options.EnableLeaderElection,
		LeaderElectionID:       "f96a6927.poddns-admission-webhook.io",
	})

	if err != nil {
		webhookLogger.Error(err, "create manager failed")
		os.Exit(1)
	}

	podInject, err := webhook.NewPodInject()
	if err != nil {
		webhookLogger.Error(err, "init podDnsOptins failed")
		os.Exit(1)
	}

	// 通过 manager 创建 webhook  server，并将自定义的处理逻辑绑定到 webhook 的 Handler 中
	if err := ctrl.NewWebhookManagedBy(mgr).
		For(&corev1.Pod{}).
		WithDefaulter(podInject).
		RecoverPanic().
		Complete(); err != nil {
		webhookLogger.Error(err, "create webhook failed")
		os.Exit(1)
	}

	webhookLogger.Info("strating manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		webhookLogger.Error(err, "start manager failed")
		os.Exit(1)
	}
}
