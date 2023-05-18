NAME ?= poddns-admission-webhook
NAMESPACE ?= default
VERSION ?= v1
IMG ?= ${NAME}:${VERSION}

##@ Help

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build

.PHONY: build
build: ## 构建二进制文件
	CGO_ENABLED=0 go build -v -o bin/${NAME} cmd/main.go

.PHONY: build-image
build-image: build ## 构建镜像
	docker build -t ${IMG} .
	docker save ${IMG} -o ${NAME}-${VERSION}.tar
	ansible k8s -m synchronize -a "src=./${NAME}-${VERSION}.tar dest=/tmp/"	
	ansible k8s -m shell -a "docker load -i /tmp/${NAME}-${VERSION}.tar"
	rm -rf ${NAME}-${VERSION}.tar

##@ TLS

.PHONY: tls
tls: ## 创建 webhook tls
	chmod +x bin/create_tls.sh
	./bin/create_tls.sh --namespace ${NAMESPACE} --service ${NAME} --secret ${NAME}-tls

##@ Install

.PHONY: install
install: ## 部署 webhook 资源
	kubectl create configmap pod-dns-options --from-file=./conf/conf.yaml --dry-run=client -o yaml > ./manifests/pod-dns-options.yaml
	sed -i "s#\( *namespace: \).*#\1${NAMESPACE}#g" ./manifests/rbac.yaml 
	sed -i "s#\(- name: \).*\(.svc\)#\1${NAME}.${NAMESPACE}\2#g" ./manifests/mutating.yaml
	sed -i "s#\(namespace: \).*#\1${NAMESPACE}#g" ./manifests/mutating.yaml
	kubectl apply -n ${NAMESPACE} -f ./manifests/


##@ Uninstall

.PHONY: uninstall
uninstall: ## 删除 webhook 资源
	kubectl delete -n ${NAMESPACE} -f ./manifests/
