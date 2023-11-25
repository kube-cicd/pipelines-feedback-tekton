package core

import (
	"github.com/kube-cicd/pipelines-feedback-core/pkgs/logging"
	"github.com/kube-cicd/pipelines-feedback-tekton/pkgs/testpkg"
	"github.com/stretchr/testify/assert"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"testing"
)

func TestFetchLogs(t *testing.T) {
	_ = testpkg.Kubectl([]string{"delete", "-f", "../../test_data/test-run.yaml"})
	assert.Nil(t, testpkg.Kubectl([]string{"apply", "-f", "../../test_data/test-run.yaml"}))
	testpkg.WaitForPipelineFinishedByName("default", "test-run")

	run := v1.PipelineRun{}
	run.Name = "test-run"
	run.Namespace = "default"

	prp := PipelineRunProvider{logger: logging.CreateLogger(true)}
	assert.Contains(t, prp.fetchLogs(&run), "Bread for you")
}
