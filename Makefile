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
