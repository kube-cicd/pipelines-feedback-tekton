package core

import (
	"github.com/kube-cicd/pipelines-feedback-core/pkgs/controller"
	"github.com/kube-cicd/pipelines-feedback-core/pkgs/store"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

func CreateTektonController() *controller.GenericController {
	return &controller.GenericController{
		PipelineInfoProvider: &PipelineRunProvider{},
		ObjectType:           &v1.PipelineRun{},
		Store:                store.Operator{Store: store.NewMemory()},
		// todo: schema provider
	}
}
