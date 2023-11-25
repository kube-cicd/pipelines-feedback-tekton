package testpkg

import (
	"github.com/sirupsen/logrus"
	"os/exec"
)

func Kubectl(argv []string) error {
	proc := exec.Command("kubectl", argv...)

	waitErr := proc.Wait()
	out, _ := proc.CombinedOutput()
	logrus.Println(string(out))
	if waitErr != nil {
		return waitErr
	}
	return nil
}

func WaitForPipelineFinishedByName(ns string, name string) {
	Kubectl([]string{"wait", "--for=condition=Succeeded", "pipelinerun", "-n", ns, name})
}

func WaitForPipelineFinishedByLabel(ns string, label string) {
	Kubectl([]string{"wait", "--for=condition=Succeeded", "pipelinerun", "-n", ns, "-l", label})
}
