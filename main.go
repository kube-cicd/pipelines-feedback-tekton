package main

import (
	"github.com/kube-cicd/pipelines-feedback-core/pkgs/app"
	"github.com/kube-cicd/pipelines-feedback-core/pkgs/cli"
	"github.com/kube-cicd/pipelines-feedback-core/pkgs/controller"
	"github.com/kube-cicd/pipelines-feedback-tekton/pkgs/core"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"os"
)

func main() {

	pfcApp := app.PipelinesFeedbackApp{
		JobController:    core.CreateTektonController(),
		ConfigController: &controller.ConfigurationController{},
		KubernetesSchemeSetters: []app.SchemeSetter{
			v1.AddToScheme,
		},
	}
	cmd := cli.NewRootCommand(&pfcApp)
	args := os.Args
	if args != nil {
		args = args[1:]
		cmd.SetArgs(args)
	}
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
