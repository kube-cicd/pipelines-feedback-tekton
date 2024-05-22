package core

import (
	"bytes"
	"context"
	"time"

	"github.com/kube-cicd/pipelines-feedback-core/pkgs/config"
	"github.com/kube-cicd/pipelines-feedback-core/pkgs/contract"
	"github.com/kube-cicd/pipelines-feedback-core/pkgs/contract/wiring"
	"github.com/kube-cicd/pipelines-feedback-core/pkgs/k8s"
	"github.com/kube-cicd/pipelines-feedback-core/pkgs/logging"
	"github.com/kube-cicd/pipelines-feedback-core/pkgs/provider"
	"github.com/kube-cicd/pipelines-feedback-core/pkgs/store"
	"github.com/kube-cicd/pipelines-feedback-core/pkgs/templating"
	"github.com/pkg/errors"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/log"
	"github.com/tektoncd/cli/pkg/options"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	v1Client "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/typed/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/apis"
)

type PipelineRunProvider struct {
	store        *store.Operator
	logger       *logging.InternalLogger
	confProvider config.ConfigurationProviderInterface
	client       v1Client.TektonV1Interface
	restConfig   *rest.Config
}

func (prp *PipelineRunProvider) InitializeWithContext(sc *wiring.ServiceContext) error {
	client, err := v1Client.NewForConfig(sc.KubeConfig)
	if err != nil {
		return errors.Wrap(err, "cannot initialize PipelineRunProvider")
	}
	prp.SetClient(client)

	prp.store = sc.Store
	prp.logger = sc.Log
	prp.confProvider = sc.Config
	prp.restConfig = sc.KubeConfig

	return nil
}

func (prp *PipelineRunProvider) SetClient(client v1Client.TektonV1Interface) {
	prp.client = client
}

func (prp *PipelineRunProvider) fetchTaskRuns(ctx context.Context, pipelineRun *v1.PipelineRun) (map[string]v1.TaskRun, error) {
	result := make(map[string]v1.TaskRun, 0)

	// finds all tasks matching this PipelineRun
	children, retrieveErr := prp.client.TaskRuns(pipelineRun.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "tekton.dev/pipelineRun=" + pipelineRun.Name,
	})
	if retrieveErr != nil {
		return result, errors.New("cannot retrieve TaskRuns by label")
	}
	for _, task := range children.Items {
		result[task.Name] = task
	}
	return result, nil
}

// ReceivePipelineInfo is tracking tekton.dev/v1, kind: PipelineRun type objects
func (prp *PipelineRunProvider) ReceivePipelineInfo(ctx context.Context, name string, namespace string, log *logging.InternalLogger) (contract.PipelineInfo, error) {
	globalCfg := prp.confProvider.FetchGlobal("global")

	pipelineRun, err := prp.client.PipelineRuns(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return contract.PipelineInfo{}, errors.Wrap(err, "cannot fetch Tekton PipelineRun")
	}

	// validate
	if ok, err := k8s.HasUsableAnnotations(pipelineRun.ObjectMeta); !ok {
		if err != nil {
			return contract.PipelineInfo{}, err
		}
		return contract.PipelineInfo{}, errors.New(provider.ErrNotMatched)
	}

	// create scm context
	scm, scmErr := k8s.CreateJobContextFromKubernetesAnnotations(pipelineRun.ObjectMeta)
	if scmErr != nil {
		return contract.PipelineInfo{}, errors.Wrap(scmErr, "cannot create scm context from a PipelineRun")
	}

	// start time
	var startTime time.Time
	if pipelineRun.HasStarted() && pipelineRun.Status.StartTime != nil {
		startTime = pipelineRun.Status.StartTime.Time
	}

	// stages
	stages, fetchErr := prp.collectStatus(ctx, pipelineRun, log)
	if fetchErr != nil {
		return contract.PipelineInfo{}, errors.Wrap(fetchErr, "cannot fetch stages list")
	}

	dashboardUrl, dashboardTplErr := templating.TemplateDashboardUrl(
		globalCfg.Get("dashboard-url"), pipelineRun, pipelineRun.TypeMeta,
	)
	if dashboardTplErr != nil {
		log.Warningf("Cannot render dashboard template URL '%s': '%s'", dashboardUrl, dashboardTplErr.Error())
	}

	pi := contract.NewPipelineInfo(
		scm,
		namespace,
		receivePipelineName(pipelineRun),
		pipelineRun.Name,
		startTime,
		stages,
		labels.Set(pipelineRun.GetLabels()),
		labels.Set(pipelineRun.GetAnnotations()),
		&globalCfg,
		contract.PipelineInfoWithUrl(dashboardUrl),
		contract.PipelineInfoWithLogsCollector(func() string { return prp.fetchLogs(pipelineRun) }),
	)
	return *pi, nil
}

// fetchLogs is using tkn cli client code to fetch logs from a PipelineRun
func (prp *PipelineRunProvider) fetchLogs(pipelineRun *v1.PipelineRun) string {
	params := &cli.TektonParams{}
	params.SetNamespace(pipelineRun.Namespace)
	params.SetNoColour(true)
	_, _ = params.Clients(prp.restConfig)

	opts := options.NewLogOptions(params)
	opts.Limit = 64
	opts.PipelineRunName = pipelineRun.Name

	lr, err := log.NewReader(log.LogTypePipeline, opts)
	if err != nil {
		prp.logger.Errorf("Cannot open logs: %s", err.Error())
		return ""
	}
	logC, errC, err := lr.Read()
	if err != nil {
		prp.logger.Errorf("Cannot read logs: %s", err.Error())
		return ""
	}

	buf := new(bytes.Buffer)
	log.NewWriter(log.LogTypePipeline, opts.Prefixing).Write(&cli.Stream{Out: buf}, logC, errC)
	return buf.String()
}

func (prp *PipelineRunProvider) collectStatus(ctx context.Context, pipelineRun *v1.PipelineRun, log *logging.InternalLogger) ([]contract.PipelineStage, error) {
	// Collect all tasks in valid order
	orderedTasks := make([]contract.PipelineStage, 0)

	if pipelineRun.Status.PipelineSpec == nil {
		return []contract.PipelineStage{}, errors.New("Pipeline is not ready yet - " +
			"we should wait a little bit until it not be picked by Tekton Controller")
	}

	for _, task := range pipelineRun.Status.PipelineSpec.Tasks {
		orderedTasks = append(orderedTasks, contract.PipelineStage{
			Name:   task.Name,
			Status: contract.PipelinePending,
		})
	}

	// Map 'PipelineTaskName' to TaskRun
	// Missing entries should be described as 'pending' (not created yet)
	mapped := make(map[string]string, 0)
	for _, task := range pipelineRun.Status.ChildReferences {
		mapped[task.PipelineTaskName] = task.Name
	}

	// Fetch TaskRuns associated with the PipelineRun
	pipelineTasks, fetchErr := prp.fetchTaskRuns(ctx, pipelineRun)
	if fetchErr != nil {
		return orderedTasks, errors.Wrap(fetchErr, "cannot fetch TaskRuns")
	}

	for num, task := range orderedTasks {
		// Skipped = on a "skipped tasks list" + TaskRun does not exist
		if isTaskSkipped(pipelineRun, task.Name) {
			orderedTasks[num].Status = contract.PipelineSkipped
			continue
		}

		taskRunName, exists := mapped[task.Name]
		if !exists {
			log.Debugf("TaskRun for task '%s' does not exist at all. Status = pending", task.Name)
			task.Status = contract.PipelinePending
			continue
		}

		taskRun, taskRunExists := pipelineTasks[taskRunName]
		if !taskRunExists {
			log.Debugf("TaskRun %s does not exist. Status = pending", taskRunName)
			task.Status = contract.PipelinePending
			continue
		}

		orderedTasks[num].Status = translateTaskStatus(&taskRun)
		log.Debugf("TaskRun '%s' status '%s'", taskRunName, orderedTasks[num].Status)
	}
	return orderedTasks, nil
}

func receivePipelineName(pipelineRun *v1.PipelineRun) string {
	pipelineName := ""
	if pipelineRun.Spec.PipelineRef != nil && pipelineRun.Spec.PipelineRef.Name != "" {
		pipelineName = pipelineRun.Spec.PipelineRef.Name
	} else {
		pipelineName = pipelineRun.Spec.PipelineSpec.DisplayName
	}

	// fallback to Tekton label
	if pipelineName == "" {
		if nameFromLabel, exists := pipelineRun.ObjectMeta.Labels["tekton.dev/pipeline"]; exists {
			return nameFromLabel
		}

		return pipelineRun.Name
	}
	return pipelineName
}

func isTaskSkipped(pipelineRun *v1.PipelineRun, taskPipelineName string) bool {
	for _, skippedTask := range pipelineRun.Status.SkippedTasks {
		if skippedTask.Name == taskPipelineName {
			return true
		}
	}
	return false
}

func translateTaskStatus(task *v1.TaskRun) contract.Status {
	if task.IsCancelled() {
		print("IsCancelled")
		return contract.PipelineCancelled
	}
	finish := task.Status.GetCondition(apis.ConditionSucceeded)
	if finish == nil {
		return contract.PipelinePending
	}
	switch finish.Reason {
	case "Pending":
		return contract.PipelinePending
	case "Succeeded":
		return contract.PipelineSucceeded
	case "Running":
		return contract.PipelineRunning
	case "ExceededResourceQuota":
		return contract.PipelinePending
	default:
		return contract.PipelineFailed
	}
}
