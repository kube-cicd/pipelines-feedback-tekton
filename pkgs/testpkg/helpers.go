package testpkg

import (
	"os/exec"
	"time"

	"github.com/sirupsen/logrus"
)

func Kubectl(argv []string) error {
	proc := exec.Command("kubectl", argv...)
	_ = proc.Start()
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
	time.Sleep(2 * time.Second)
}

func WaitForPipelineFinishedByLabel(ns string, label string) {
	Kubectl([]string{"wait", "--for=condition=Succeeded", "pipelinerun", "-n", ns, "-l", label})
	time.Sleep(2 * time.Second)
}
