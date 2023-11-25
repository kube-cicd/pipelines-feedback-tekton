fetch-common:
	@cd ../.. && curl https://github.com/kube-cicd/pipelines-feedback-core/archive/refs/heads/main.tar.gz -L | tar -zxvf - pipelines-feedback-core-main/charts/pipelines-feedback-common; \
	rm -rf charts/pipelines-feedback-common; \
	mv pipelines-feedback-core-main/charts/pipelines-feedback-common charts/pipelines-feedback-common; \
	rm -rf pipelines-feedback-core-main

build: fetch-common
	cd .. && rsync -av ./pipelines-feedback-common/* ./tekton-chart/ --exclude Makefile --exclude README.md.gotmpl
	cd ../tekton-chart && yq e -i '.name= "batchv1-chart"' Chart.yaml
	yq '. *= load("values.override.yaml")' values.yaml > values.copy && mv values.copy values.yaml
