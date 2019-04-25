package k8s

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"code.cloudfoundry.org/eirini/metrics"
	"code.cloudfoundry.org/eirini/route"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

type MetricsCollector struct {
	work          chan<- []metrics.Message
	metricsClient metricsclientset.Interface
	scheduler     route.TaskScheduler
	namespace     string
}

func NewMetricsCollector(work chan []metrics.Message, scheduler route.TaskScheduler, metricsClient metricsclientset.Interface) *MetricsCollector {
	return &MetricsCollector{
		work:          work,
		metricsClient: metricsClient,
		scheduler:     scheduler,
	}
}

func (c *MetricsCollector) Start() {
	c.scheduler.Schedule(func() error {
		_, err := c.metricsClient.MetricsV1beta1().PodMetricses(c.namespace).List(metav1.ListOptions{})
		if err != nil {
			return err //todo: wrap
		}
		return nil
	})
}

func (c *MetricsCollector) convertMetricsList(podMetrics *v1beta1.PodMetricsList) ([]metrics.Message, error) {
	//for _, metric := range podMetrics.Items {
	//if len(metric.Containers) == 0 {
	//continue
	//}
	//container := metric.Containers[0]
	//_, indexID, err := util.ParseAppNameAndIndex(metric.Metadata.Name)
	//if err != nil {
	//return nil, err
	//}
	//cpuValue, err := extractValue(container.Usage.CPU)
	//if err != nil {
	//return nil, errors.Wrap(err, "Failed to convert cpu value")
	//}
	//memoryValue, err := extractValue(container.Usage.Memory)
	//if err != nil {
	//return nil, errors.Wrap(err, "Failed to convert memory values")
	//}

	//pod, err := c.podClient.Get(metric.Metadata.Name, meta.GetOptions{})
	//if err != nil {
	//return []metrics.Message{}, err
	//}

	//messages = append(messages, metrics.Message{
	//AppID:       pod.Labels["guid"],
	//IndexID:     strconv.Itoa(indexID),
	//CPU:         convertCPU(cpuValue),
	//Memory:      convertMemory(memoryValue),
	//MemoryQuota: 10,
	//Disk:        42000000,
	//DiskQuota:   10,
	//})
	//}
	//return messages, nil
	return []metrics.Message{}, nil
}

func extractValue(metric string) (float64, error) {
	re := regexp.MustCompile("[a-zA-Z]+")
	match := re.FindStringSubmatch(metric)
	if len(match) == 0 {
		f, err := strconv.ParseFloat(metric, 64)
		return f, errors.Wrap(err, fmt.Sprintf("failed to parse metric %s", metric))
	}

	unit := match[0]
	valueStr := strings.Trim(metric, unit)

	return strconv.ParseFloat(valueStr, 64)
}

func convertCPU(cpuUsage float64) float64 {
	return cpuUsage / 1000
}

func convertMemory(memoryUsage float64) float64 {
	return memoryUsage * 1024
}
