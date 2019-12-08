package collector

import (
	"fmt"
	"github.com/dlopes7/aix-prometheus-exporter/c_api"
	"github.com/prometheus/client_golang/prometheus"
)

type statCollector struct {
	cpu *prometheus.Desc
}

func init() {
	registerCollector("cpu", true, NewCPUCollector)
}

func NewCPUCollector() (Collector, error) {
	return &statCollector{
		cpu: nodeCPUSecondsDesc,
	}, nil
}

func (c *statCollector) Update(ch chan<- prometheus.Metric) error {
	var fieldsCount = 4
	cpuFields := []string{"user", "sys", "wait", "idle"}

	cpuTimes, err := c_api.GetAIXCPUTimes()
	if err != nil {
		return err
	}

	for i, value := range cpuTimes {
		cpux := fmt.Sprintf("CPU %d", i/fieldsCount)
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, value, cpux, cpuFields[i%fieldsCount])
	}

	return nil
}
