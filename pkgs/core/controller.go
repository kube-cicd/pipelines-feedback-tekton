package core

import (
	"github.com/kube-cicd/pipelines-feedback-core/pkgs/controller"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

func CreateTektonController() *controller.GenericController {
	return &controller.GenericController{
		PipelineInfoProvider: &PipelineRunProvider{},
		ObjectType:           &v1.PipelineRun{},
	}
}
