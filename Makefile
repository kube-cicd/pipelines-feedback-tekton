SUDO=
ENV_CLUSTER_NAME=pft

.PHONY: all
all: build

.EXPORT_ALL_VARIABLES:
PATH = $(LOCALBIN):$(shell echo $$PATH)

.PHONY: build
build: fmt vet ## Build manager binary.
	@mkdir -p $(LOCALBIN)
	go build -o $(LOCALBIN)/pipelines-feedback-tekton main.go

run:
	./.build/pipelines-feedback-tekton --debug

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
	export PATH=$$PATH:~/go/bin:$$GOROOT/bin:$$(pwd)/.build; go test -v ./... -covermode=count -coverprofile=coverage.out 2>&1 | go-junit-report -set-exit-code -out junit.xml -iocopy

k3d:
	(${SUDO} docker ps | grep k3d-${ENV_CLUSTER_NAME}-server-0 > /dev/null 2>&1) || ${SUDO} k3d cluster create ${ENV_CLUSTER_NAME} --registry-create ${ENV_CLUSTER_NAME}-registry:0.0.0.0:5000 --agents 0
	k3d kubeconfig merge ${ENV_CLUSTER_NAME}

k3d-install-tekton:
	export KUBECONFIG=~/.k3d/kubeconfig-${ENV_CLUSTER_NAME}.yaml; \
	kubectl create ns tekton-pipelines || true; \
	kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/previous/v0.44.4/release.yaml
