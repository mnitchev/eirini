package rootfs

import (
	"encoding/json"
	"fmt"

	"code.cloudfoundry.org/eirini/route"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
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

type patch struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
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
	fmt.Println(selector)
	opts := meta.ListOptions{LabelSelector: selector}
	ss, err := s.statefulSets().List(opts)
	if err != nil {
		fmt.Println("Cant find pods")
		return err
	}

	payload := []patch{{
		Op:    "add",
		Path:  "/spec/template/metadata/labels/eirinifs-digest",
		Value: string(version),
	}}
	payloadBytes, _ := json.Marshal(payload)

	for _, statefulset := range ss.Items {
		_, err = s.statefulSets().Patch(statefulset.Name, apimachinerytypes.JSONPatchType, payloadBytes)
		if err != nil {
			fmt.Println("Cant update pods")
			return err
		}
	}
	return nil
}

func (s *Sink) statefulSets() types.StatefulSetInterface {
	return s.Client.AppsV1beta2().StatefulSets(s.Namespace)
}
