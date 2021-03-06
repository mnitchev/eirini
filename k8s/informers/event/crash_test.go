package event_test

import (
	"fmt"
	"time"

	"code.cloudfoundry.org/eirini/events"
	. "code.cloudfoundry.org/eirini/k8s/informers/event"
	"code.cloudfoundry.org/eirini/models/cf"
	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/runtimeschema/cc_messages"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
)

var _ = Describe("Event", func() {

	const namespace = "milkyway"

	var (
		client        kubernetes.Interface
		crashInformer *CrashInformer

		reportChan      chan events.CrashReport
		informerStopper chan struct{}

		watcher               *watch.FakeWatcher
		pinky, brain, bandito *v1.Pod

		crashTime meta.Time
	)

	BeforeEach(func() {
		pinky = createPod("pinky-pod")
		brain = createPod("brain-pod")
		bandito = createStatelessPod("bandito")

		reportChan = make(chan events.CrashReport)
		informerStopper = make(chan struct{})

		client = fake.NewSimpleClientset()
		crashInformer = NewCrashInformer(client, 0, namespace, reportChan, informerStopper, lagertest.NewTestLogger("test-logger"))

		watcher = watch.NewFake()
		fakecs := client.(*fake.Clientset)
		fakecs.PrependWatchReactor("pods", testing.DefaultWatchReactor(watcher, nil))
	})

	AfterEach(func() {
		close(informerStopper)
	})

	JustBeforeEach(func() {
		go crashInformer.Start()

		watcher.Add(pinky)
		watcher.Add(brain)
		watcher.Add(bandito)
	})

	Context("When an app crashes with waiting status", func() {

		var (
			pinkyCopy   *v1.Pod
			brainCopy   *v1.Pod
			banditoCopy *v1.Pod
		)

		JustBeforeEach(func() {
			watcher.Modify(pinkyCopy)
			watcher.Modify(brainCopy)
			watcher.Modify(banditoCopy)
		})

		Context("has waiting status", func() {
			BeforeEach(func() {
				pinkyCopy = createPod("pinky-pod")
				crashTime = meta.Time{Time: time.Now()}
				pinkyCopy.Status.ContainerStatuses = []v1.ContainerStatus{
					{
						RestartCount: 3,
						State: v1.ContainerState{
							Waiting: &v1.ContainerStateWaiting{
								Reason: CrashLoopBackOff,
							},
						},
						LastTerminationState: v1.ContainerState{
							Terminated: &v1.ContainerStateTerminated{
								ExitCode:  -1,
								Reason:    "this describes how much you screwed up",
								StartedAt: crashTime,
							},
						},
					},
				}

				brainCopy = createPod("brain-pod")
				brainCopy.Status.ContainerStatuses = []v1.ContainerStatus{
					{
						State: v1.ContainerState{
							Waiting: &v1.ContainerStateWaiting{
								Reason: "sleepy",
							},
						},
					},
				}

				banditoCopy = createStatelessPod("bandito")
				banditoCopy.Name = "no-bandito"
			})

			It("should send reports the report chan", func() {
				Eventually(reportChan).Should(Receive())
			})

			It("should receive a crashed report", func() {
				Eventually(reportChan).Should(Receive(Equal(events.CrashReport{
					ProcessGUID: "pinky-pod-anno",
					AppCrashedRequest: cc_messages.AppCrashedRequest{
						Reason:          CrashLoopBackOff,
						Instance:        "pinky-pod-0",
						Index:           0,
						ExitStatus:      -1,
						ExitDescription: "this describes how much you screwed up",
						CrashCount:      3,
						CrashTimestamp:  int64(crashTime.Time.Second()),
					},
				})))
			})

			It("should not get more reports", func() {
				Eventually(reportChan).Should(Receive())
				Consistently(reportChan).ShouldNot(Receive())
			})
		})

		Context("has terminated status", func() {

			BeforeEach(func() {
				pinkyCopy = createPod("pinky-pod")
				pinkyCopy.Status.ContainerStatuses = []v1.ContainerStatus{
					{
						State: v1.ContainerState{
							Waiting: &v1.ContainerStateWaiting{
								Reason: "sleepy",
							},
						},
					},
				}

				brainCopy = createPod("brain-pod")
				crashTime = meta.Time{Time: time.Now()}
				brainCopy.Status.ContainerStatuses = []v1.ContainerStatus{
					{
						RestartCount: 8,
						State: v1.ContainerState{
							Terminated: &v1.ContainerStateTerminated{
								ExitCode:  -1,
								Reason:    "this describes how much you screwed up",
								StartedAt: crashTime,
							},
						},
					},
				}

				banditoCopy = createStatelessPod("bandito")
				banditoCopy.Name = "no-bandito"
			})

			It("should send reports the report chan", func() {
				Eventually(reportChan).Should(Receive())
			})

			It("should receive a crashed report", func() {
				Eventually(reportChan).Should(Receive(Equal(events.CrashReport{
					ProcessGUID: "brain-pod-anno",
					AppCrashedRequest: cc_messages.AppCrashedRequest{
						Reason:          "this describes how much you screwed up",
						Instance:        "brain-pod-0",
						Index:           0,
						ExitStatus:      -1,
						ExitDescription: "this describes how much you screwed up",
						CrashCount:      8,
						CrashTimestamp:  int64(crashTime.Time.Second()),
					},
				})))
			})

			It("should not get more reports", func() {
				Eventually(reportChan).Should(Receive())
				Consistently(reportChan).ShouldNot(Receive())
			})

			Context("exited normally", func() {
				BeforeEach(func() {
					brainCopy.Status.ContainerStatuses[0].State.Terminated.ExitCode = 0
				})

				It("should not send reports", func() {
					Consistently(reportChan).ShouldNot(Receive())
				})

			})

		})

	})

	Context("When a pod has no container statuses", func() {
		JustBeforeEach(func() {
			watcher.Modify(pinky)
		})

		Context("container statuses is nil", func() {
			BeforeEach(func() {
				pinky.Status.ContainerStatuses = nil
			})

			It("should not send any reports", func() {
				Consistently(reportChan).ShouldNot(Receive())
			})
		})

		Context("container statuses is empty", func() {
			BeforeEach(func() {
				pinky.Status.ContainerStatuses = []v1.ContainerStatus{}
			})

			It("should not send any reports", func() {
				Consistently(reportChan).ShouldNot(Receive())
			})
		})
	})

	Context("When a pod was just stopped or deleted", func() {
		BeforeEach(func() {
			event := v1.Event{
				InvolvedObject: v1.ObjectReference{
					Namespace: namespace,
					Name:      "pinky-pod",
				},
				Reason: "Killing",
			}
			_, clientErr := client.CoreV1().Events(namespace).Create(&event)
			Expect(clientErr).ToNot(HaveOccurred())
		})

		JustBeforeEach(func() {
			pinky.Status.ContainerStatuses = []v1.ContainerStatus{
				{
					State: v1.ContainerState{
						Terminated: &v1.ContainerStateTerminated{
							ExitCode: 1,
						},
					},
				},
			}

			watcher.Modify(pinky)
		})

		It("should not emit a crashed event", func() {
			Consistently(reportChan).ShouldNot(Receive())
		})

	})
})

func createPod(name string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: meta.ObjectMeta{
			Name: fmt.Sprintf("%s-%d", name, 0),
			Annotations: map[string]string{
				cf.ProcessGUID: fmt.Sprintf("%s-anno", name),
			},
			OwnerReferences: []meta.OwnerReference{
				{
					Kind: "StatefulSet",
					Name: "mr-stateful",
				},
			},
		},
		Status: v1.PodStatus{
			ContainerStatuses: []v1.ContainerStatus{
				{
					State: v1.ContainerState{
						Running: &v1.ContainerStateRunning{},
					},
				},
			},
		},
	}
}

func createStatelessPod(name string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: meta.ObjectMeta{
			Name: name,
		},
		Status: v1.PodStatus{
			ContainerStatuses: []v1.ContainerStatus{
				{
					State: v1.ContainerState{
						Running: &v1.ContainerStateRunning{},
					},
				},
			},
		},
	}
}
