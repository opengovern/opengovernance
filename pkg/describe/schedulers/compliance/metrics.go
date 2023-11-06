package compliance

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var ComplianceJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "kaytu_scheduler_schedule_compliance_job_total",
	Help: "Count of describe jobs in scheduler service",
}, []string{"status"})
