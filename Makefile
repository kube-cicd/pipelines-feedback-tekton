SUDO=
ENV_CLUSTER_NAME=pft
BIN_NAME=pipelines-feedback-tekton

.PHONY: all
all: build

.EXPORT_ALL_VARIABLES:
PATH = $(LOCALBIN):$(shell echo $$PATH)

.PHONY: build
build: fmt vet ## Build manager binary.
	@mkdir -p $(LOCALBIN)
	go build -o $(LOCALBIN)/${BIN_NAME} main.go

run:
	./.build/p${BIN_NAME} --debug

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

LOCALBIN ?= $(shell pwd)/.build
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

ensure-go-junit-report:
	@command -v go-junit-report || (cd /tmp && go install github.com/jstemmer/go-junit-report/v2@latest)

test: ensure-go-junit-report
	export KUBECONFIG=~/.k3d/kubeconfig-${ENV_CLUSTER_NAME}.yaml; export PATH=$$PATH:~/go/bin:$$GOROOT/bin:$$(pwd)/.build; go test -v ./... -covermode=count -coverprofile=coverage.out 2>&1 | go-junit-report -set-exit-code -out junit.xml -iocopy

tools-install:
	wget https://github.com/k3d-io/k3d/releases/download/v5.6.0/k3d-linux-amd64 -O .build/k3d 2>/dev/null
	wget https://dl.k8s.io/release/v1.28.4/bin/linux/amd64/kubectl -O .build/kubectl 2>/dev/null
	@chmod +x .build/k3d .build/kubectl

k3d: tools-install
	(${SUDO} docker ps | grep k3d-${ENV_CLUSTER_NAME}-server-0 > /dev/null 2>&1) || ${SUDO} ./.build/k3d cluster create ${ENV_CLUSTER_NAME} --registry-create ${ENV_CLUSTER_NAME}-registry:0.0.0.0:5000 --agents 0
	mkdir -p ~/.k3d; path=$$(./.build/k3d kubeconfig merge ${ENV_CLUSTER_NAME}); cp $$path ~/.k3d/kubeconfig-${ENV_CLUSTER_NAME}.yaml

k3d-install-tekton:
	export KUBECONFIG=~/.k3d/kubeconfig-${ENV_CLUSTER_NAME}.yaml; \
	./.build/kubectl create ns tekton-pipelines || true; \
	./.build/kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/previous/v0.44.4/release.yaml; \
	./utils/test/wait-for-pods.sh -l app.kubernetes.io/part-of=tekton-pipelines -n tekton-pipelines
