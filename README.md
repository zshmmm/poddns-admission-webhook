# webhook

对新建 pod 注入 `DnsConfig` 相关配置。

## 部署
默认资源部署到 `default` namespace 下，可以通过修改 `Makefile` 文件中的 `NAMESPACE ?= default` 变量部署到指定的 namespace。
目标 namespace 必须存在。

1. 补齐 mutating 认证 CA

```bash
# 重命名文件 mutating-tmp
cp ./manifests/mutating-tmp ./manifests/mutating.yaml

# 编辑 mutating.yaml 文件按照说明填充 caBundle: 字段
```

2. 生成 webhook TLS 证书

```bash
make tls
```

3. 编译打包镜像

```bash
# 本地测试环境将镜像同步到集群节点，使用的话需要镜像推送到镜像仓库，并更新 ./manifests/deployment.yaml 文件中的webhook镜像地址
make build-image
```

4. 调整参数
根据实际情况修改 ./conf/conf.yaml 配置

5. 部署

```bash
make install
```