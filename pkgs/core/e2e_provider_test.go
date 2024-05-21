package core

import (
	"context"
	"testing"

	"github.com/kube-cicd/pipelines-feedback-core/pkgs/contract/wiring"
	"github.com/kube-cicd/pipelines-feedback-core/pkgs/fake"
	"github.com/kube-cicd/pipelines-feedback-core/pkgs/logging"
	"github.com/kube-cicd/pipelines-feedback-core/pkgs/store"
	"github.com/kube-cicd/pipelines-feedback-tekton/pkgs/testpkg"

	"github.com/stretchr/testify/assert"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonFake "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	duckv1 "knative.dev/pkg/apis/duck/v1"
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

// TestSkippedTask is checking if task marked as skipped is shown as skipped. Previously it was recognized as "pending"
func TestSkippedTask(t *testing.T) {
	prp := PipelineRunProvider{logger: logging.CreateLogger(true)}
	prp.InitializeWithContext(&wiring.ServiceContext{
		Recorder:     record.NewFakeRecorder(3),
		Config:       &fake.FakeConfigurationProvider{},
		KubeConfig:   &rest.Config{},
		Log:          logging.NewInternalLogger(),
		Store:        &store.Operator{Store: store.NewMemory()},
		ConfigSchema: &fake.NullValidator{},
	})

	tekton := tektonFake.NewSimpleClientset().TektonV1()
	prp.SetClient(tekton)

	spec := &v1.PipelineSpec{
		DisplayName: "fast-feedback",
		Tasks: []v1.PipelineTask{
			{Name: "skipped"},
		},
	}
	tekton.PipelineRuns("default").Create(context.TODO(), &v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			Labels: map[string]string{
				"pipelinesfeedback.keskad.pl/enabled": "true",
			},
			Annotations: map[string]string{
				"pipelinesfeedback.keskad.pl/https-repo-url": "https://github.com/keskad/jx-gitops",
				"pipelinesfeedback.keskad.pl/ref":            "refs/heads/test-pr",
				"pipelinesfeedback.keskad.pl/commit":         "76ea7c746d4e4ac42c44bf72946d3b0d399553dd",
				"pipelinesfeedback.keskad.pl/pr-id":          "2",
			},
		},
		Status: v1.PipelineRunStatus{
			Status: duckv1.Status{},
			PipelineRunStatusFields: v1.PipelineRunStatusFields{
				PipelineSpec: spec,
				SkippedTasks: []v1.SkippedTask{
					{Name: "skipped", Reason: "ups"},
				},
			},
		},
		Spec: v1.PipelineRunSpec{
			PipelineSpec: spec,
		},
	}, metav1.CreateOptions{})

	info, err := prp.ReceivePipelineInfo(context.TODO(), "test", "default", logging.NewInternalLogger())

	assert.Equal(t, "was skipped", info.GetStages()[0].Status.AsHumanReadableDescription())
	assert.Equal(t, "succeeded", info.GetStatus().AsHumanReadableDescription())
	assert.Nil(t, err)
}
