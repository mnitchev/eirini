package k8s_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "code.cloudfoundry.org/eirini/k8s"
	"code.cloudfoundry.org/eirini/metrics"
	"code.cloudfoundry.org/eirini/route/routefakes"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	testcore "k8s.io/client-go/testing"
	metricsv1beta1api "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsfake "k8s.io/metrics/pkg/client/clientset/versioned/fake"
)

var _ = FDescribe("Metrics", func() {

	var (
		collector     *MetricsCollector
		work          chan []metrics.Message
		metricsClient *metricsfake.Clientset
		scheduler     *routefakes.FakeTaskScheduler
	)

	BeforeEach(func() {
		metricsClient = metricsfake.NewSimpleClientset()
	})

	JustBeforeEach(func() {
		scheduler = new(routefakes.FakeTaskScheduler)
		work = make(chan []metrics.Message, 1)
		collector = NewMetricsCollector(work, scheduler, metricsClient)
	})

	Context("When collecting metrics", func() {

		var err error

		BeforeEach(func() {

			expectedMetrics := metricsv1beta1api.PodMetricsList{
				Items: []metricsv1beta1api.PodMetrics{
					{
						Containers: []metricsv1beta1api.ContainerMetrics{
							{
								Usage: v1.ResourceList{
									v1.ResourceCPU:    resource.MustParse("420001m"),
									v1.ResourceMemory: resource.MustParse("42Ki"),
								},
							},
						},
					},
				},
			}

			metricsClient.AddReactor("list", "pods", func(action testcore.Action) (handled bool, ret runtime.Object, err error) {
				return true, &expectedMetrics, nil
			})
		})

		JustBeforeEach(func() {
			collector.Start()
			task := scheduler.ScheduleArgsForCall(0)
			err = task()
		})

		It("should not return an error", func() {
			Expect(err).ToNot(HaveOccurred())
		})

		It("should send the received metrics", func() {
			Eventually(work).Should(Receive(Equal([]metrics.Message{
				{
					AppID:       "app-guid",
					IndexID:     "9000",
					CPU:         420,
					Memory:      430080,
					MemoryQuota: 10,
					Disk:        42000000,
					DiskQuota:   10,
				},
			})))
		})

		Context("there are no items", func() {

			BeforeEach(func() {
			})

			It("should not return an error", func() {
				Expect(err).ToNot(HaveOccurred())
			})

			It("should not send anything", func() {
				Consistently(work).ShouldNot(Receive())
			})
		})

		Context("there are no containers", func() {

			BeforeEach(func() {
			})

			It("should not return an error", func() {
				Expect(err).ToNot(HaveOccurred())
			})

			It("should not send anything", func() {
				Consistently(work).ShouldNot(Receive())
			})
		})

		Context("memory metric does not have a unit", func() {
			BeforeEach(func() {
			})

			It("should return not an error", func() {
				Expect(err).ToNot(HaveOccurred())
			})

			It("should send the received metrics", func() {
				Eventually(work).Should(Receive(Equal([]metrics.Message{
					{
						AppID:       "app-guid",
						IndexID:     "9000",
						CPU:         420,
						Memory:      430080,
						MemoryQuota: 10,
						Disk:        42000000,
						DiskQuota:   10,
					},
				})))
			})
		})

		Context("pod name doesn't have an index (eg staging tasks)", func() {
			BeforeEach(func() {
			})

			It("should return an error", func() {
				Expect(err).To(HaveOccurred())
			})

			It("should not send a message", func() {
				Expect(work).ShouldNot(Receive())
			})
		})

		Context("metrics source responds with an error", func() {

			BeforeEach(func() {
			})

			It("should return an error", func() {
				Expect(err).To(MatchError(ContainSubstring("metrics source responded with code: 400")))
			})
		})
	})
})
