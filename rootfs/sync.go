package rootfs

import (
	"fmt"

	"code.cloudfoundry.org/eirini/route"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	types "k8s.io/client-go/kubernetes/typed/apps/v1beta2"
)

const VersionLabel = "eirinifs-digest"

type Sink struct {
	Digester  *Digester
	Client    kubernetes.Interface
	Namespace string
	Scheduler route.TaskScheduler
}

func (s *Sink) Watch() {
	s.Scheduler.Schedule(s.poll)
}

func (s *Sink) poll() error {
	version, err := s.Digester.Get()
	if err != nil {
		return err
	}

	selector := fmt.Sprintf("%s notin (%s)", VersionLabel, version)
	opts := meta.ListOptions{LabelSelector: selector}
	ss, err := s.statefulSets().List(opts)
	if err != nil {
		return err
	}

	for _, statefulset := range ss.Items {
		statefulset.Spec.Template.Labels[VersionLabel] = string(version)
		_, err = s.statefulSets().Update(&statefulset)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Sink) statefulSets() types.StatefulSetInterface {
	return s.Client.AppsV1beta2().StatefulSets(s.Namespace)
}
