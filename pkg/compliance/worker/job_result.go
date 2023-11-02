package worker

import complianceapi "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"

type JobResult struct {
	Job    Job
	Status complianceapi.ComplianceReportJobStatus
	Error  string
}
