package worker

import "github.com/elastic/go-elasticsearch/v7"

type ComplianceReportCleanupJob struct {
	JobID uint // ComplianceReportJob ID
}

func (j ComplianceReportCleanupJob) Do(esClient *elasticsearch.Client) error {
	return nil
	//startTime := time.Now().Unix()
	//
	//ctx, cancel := context.WithTimeout(context.Background(), cleanupJobTimeout)
	//defer cancel()
	//
	//fmt.Printf("Cleaning report with compliance_report_job_id of %d from index %s\n", j.JobID, es.ComplianceReportIndex)
	//
	//query := map[string]interface{}{
	//	"query": map[string]interface{}{
	//		"match": map[string]interface{}{
	//			"report_job_id": j.JobID,
	//		},
	//	},
	//}
	//
	//indices := []string{
	//	es.ComplianceReportIndex,
	//}
	//
	//resp, err := keibi.DeleteByQuery(ctx, esClient, indices, query,
	//	esClient.DeleteByQuery.WithRefresh(true),
	//	esClient.DeleteByQuery.WithConflicts("proceed"),
	//)
	//if err != nil {
	//	DoComplianceReportCleanupJobsDuration.WithLabelValues("failure").Observe(float64(time.Now().Unix() - startTime))
	//	DoComplianceReportCleanupJobsCount.WithLabelValues("failure").Inc()
	//	return err
	//}
	//
	//if len(resp.Failures) != 0 {
	//	body, err := json.Marshal(resp)
	//	if err != nil {
	//		DoComplianceReportCleanupJobsDuration.WithLabelValues("failure").Observe(float64(time.Now().Unix() - startTime))
	//		DoComplianceReportCleanupJobsCount.WithLabelValues("failure").Inc()
	//		return err
	//	}
	//
	//	DoComplianceReportCleanupJobsDuration.WithLabelValues("failure").Observe(float64(time.Now().Unix() - startTime))
	//	DoComplianceReportCleanupJobsCount.WithLabelValues("failure").Inc()
	//	return fmt.Errorf("elasticsearch: delete by query: %s", string(body))
	//}
	//
	//fmt.Printf("Successfully delete %d reports\n", resp.Deleted)
	//DoComplianceReportCleanupJobsDuration.WithLabelValues("successful").Observe(float64(time.Now().Unix() - startTime))
	//DoComplianceReportCleanupJobsCount.WithLabelValues("successful").Inc()
	//return nil
}
