package describe

import (
	"encoding/json"
	"fmt"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	complianceapi "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report/api"

	"gitlab.com/keibiengine/keibi-engine/pkg/aws"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure"
	compliancereport "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/queue"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	JobCompletionInterval       = 1 * time.Minute
	JobSchedulingInterval       = 1 * time.Minute
	JobComplianceReportInterval = 1 * time.Minute
	JobTimeoutCheckInterval     = 15 * time.Minute
	MaxJobInQueue               = 10000
	ConcurrentDeletedSources    = 1000
)

var DescribePublishingBlocked = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "keibi",
	Subsystem: "scheduler",
	Name:      "queue_job_publishing_blocked",
	Help:      "The gauge whether publishing tasks to a queue is blocked: 0 for resumed and 1 for blocked",
}, []string{"queue_name"})

var DescribeJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "keibi",
	Subsystem: "scheduler",
	Name:      "schedule_describe_jobs_total",
	Help:      "Count of describe jobs in scheduler service",
}, []string{"status"})

var DescribeSourceJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "keibi_scheduler_schedule_describe_source_jobs_total",
	Help: "Count of describe source jobs in scheduler service",
}, []string{"status"})

var DescribeCleanupJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "keibi_scheduler_schedule_describe_cleanup_jobs_total",
	Help: "Count of describe jobs in scheduler service",
}, []string{"status"})

var DescribeCleanupSourceJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "keibi_scheduler_schedule_describe_cleanup_source_jobs_total",
	Help: "Count of describe source jobs in scheduler service",
}, []string{"status"})

var ComplianceJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "keibi_scheduler_schedule_compliance_job_total",
	Help: "Count of describe jobs in scheduler service",
}, []string{"status"})

var ComplianceSourceJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "keibi_scheduler_schedule_compliance_source_job_total",
	Help: "Count of describe source jobs in scheduler service",
}, []string{"status"})

type Scheduler struct {
	id         string
	db         Database
	httpServer *HttpServer

	// describeJobQueue is used to publish describe jobs to be performed by the workers.
	describeJobQueue queue.Interface
	// describeJobResultQueue is used to consume the describe job results returned by the workers.
	describeJobResultQueue queue.Interface
	// describeCleanupJobQueue is used to publish describe cleanup jobs to be performed by the workers.
	describeCleanupJobQueue queue.Interface

	// sourceQueue is used to consume source updates by the onboarding service.
	sourceQueue queue.Interface

	complianceReportJobQueue        queue.Interface
	complianceReportJobResultQueue  queue.Interface
	complianceReportCleanupJobQueue queue.Interface

	// watch the deleted source
	deletedSources chan string

	logger *zap.Logger
}

func InitializeScheduler(
	id string,
	rabbitMQUsername string,
	rabbitMQPassword string,
	rabbitMQHost string,
	rabbitMQPort int,
	describeJobQueueName string,
	describeJobResultQueueName string,
	describeCleanupJobQueueName string,
	complianceReportJobQueueName string,
	complianceReportJobResultQueueName string,
	complianceReportCleanupJobQueueName string,
	sourceQueueName string,
	postgresUsername string,
	postgresPassword string,
	postgresHost string,
	postgresPort string,
	postgresDb string,
	httpServerAddress string,
	vaultAddress string,
	vaultRoleName string,
	vaultToken string,
) (s *Scheduler, err error) {
	if id == "" {
		return nil, fmt.Errorf("'id' must be set to a non empty string")
	}

	s = &Scheduler{
		id:             id,
		deletedSources: make(chan string, ConcurrentDeletedSources),
	}
	defer func() {
		if err != nil && s != nil {
			s.Stop()
		}
	}()

	s.logger, err = zap.NewProduction()
	if err != nil {
		return nil, err
	}

	s.logger.Info("Initializing the scheduler")

	qCfg := queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = describeJobQueueName
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = s.id
	describeQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Connected to the describe jobs queue", zap.String("queue", describeJobQueueName))
	s.describeJobQueue = describeQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = describeJobResultQueueName
	qCfg.Queue.Durable = true
	qCfg.Consumer.ID = s.id
	describeResultsQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Connected to the describe job results queue", zap.String("queue", describeJobResultQueueName))
	s.describeJobResultQueue = describeResultsQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = describeCleanupJobQueueName
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = s.id
	describeCleanupJobQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Connected to the describe cleanup job queue", zap.String("queue", describeCleanupJobQueueName))
	s.describeCleanupJobQueue = describeCleanupJobQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = complianceReportCleanupJobQueueName
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = s.id
	complianceReportCleanupJobQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Connected to the complianceReport cleanup job queue", zap.String("queue", complianceReportCleanupJobQueueName))
	s.complianceReportCleanupJobQueue = complianceReportCleanupJobQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = sourceQueueName
	qCfg.Queue.Durable = true
	qCfg.Consumer.ID = s.id
	sourceEventsQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Connected to the source events queue", zap.String("queue", sourceQueueName))
	s.sourceQueue = sourceEventsQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = complianceReportJobQueueName
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = s.id
	complianceReportJobsQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Connected to the compliance report jobs queue", zap.String("queue", complianceReportJobQueueName))
	s.complianceReportJobQueue = complianceReportJobsQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = complianceReportJobResultQueueName
	qCfg.Queue.Durable = true
	qCfg.Consumer.ID = s.id
	complianceReportJobsResultQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Connected to the compliance report jobs result queue", zap.String("queue", complianceReportJobResultQueueName))
	s.complianceReportJobResultQueue = complianceReportJobsResultQueue

	dsn := fmt.Sprintf(`host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=GMT`,
		postgresHost,
		postgresPort,
		postgresUsername,
		postgresPassword,
		postgresDb,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	s.logger.Info("Connected to the postgres database: ", zap.String("db", postgresDb))
	s.db = Database{orm: db}

	s.httpServer = NewHTTPServer(httpServerAddress, s.db)

	return s, nil
}

func (s *Scheduler) Run() error {
	err := s.db.Initialize()
	if err != nil {
		return err
	}

	go s.RunDescribeJobCompletionUpdater()
	go s.RunDescribeJobScheduler()
	go s.RunDescribeCleanupJobScheduler()
	go s.RunComplianceReportScheduler()

	// In order to have history of reports, we won't clean up compliance reports for now.
	//go s.RunComplianceReportCleanupJobScheduler()

	go func() {
		s.logger.Fatal("SourceEvent consumer exited", zap.Error(s.RunSourceEventsConsumer()))
	}()

	go func() {
		s.logger.Fatal("DescribeJobResult consumer exited", zap.Error(s.RunDescribeJobResultsConsumer()))
	}()

	go func() {
		s.logger.Fatal("ComplianceReportJobResult consumer exited", zap.Error(s.RunComplianceReportJobResultsConsumer()))
	}()

	return httpserver.RegisterAndStart(s.logger, s.httpServer.Address, s.httpServer)
}

func (s *Scheduler) RunDescribeJobCompletionUpdater() {
	t := time.NewTicker(JobCompletionInterval)
	defer t.Stop()

	for ; ; <-t.C {
		results, err := s.db.QueryInProgressDescribedSourceJobGroupByDescribeResourceJobStatus()
		if err != nil {
			s.logger.Error("Failed to find DescribeSourceJobs", zap.Error(err))
			continue
		}

		jobIDToStatus := make(map[uint]map[api.DescribeResourceJobStatus]int)
		for _, v := range results {
			if _, ok := jobIDToStatus[v.DescribeSourceJobID]; !ok {
				jobIDToStatus[v.DescribeSourceJobID] = map[api.DescribeResourceJobStatus]int{
					api.DescribeResourceJobCreated:   0,
					api.DescribeResourceJobQueued:    0,
					api.DescribeResourceJobFailed:    0,
					api.DescribeResourceJobSucceeded: 0,
				}
			}

			jobIDToStatus[v.DescribeSourceJobID][v.DescribeResourceJobStatus] = v.DescribeResourceJobCount
		}

		for id, status := range jobIDToStatus {
			// If any CREATED or QUEUED, job is still in progress
			if status[api.DescribeResourceJobCreated] > 0 ||
				status[api.DescribeResourceJobQueued] > 0 {
				continue
			}

			// If any FAILURE, job is completed with failure
			if status[api.DescribeResourceJobFailed] > 0 {
				err := s.db.UpdateDescribeSourceJob(id, api.DescribeSourceJobCompletedWithFailure)
				if err != nil {
					s.logger.Error("Failed to update DescribeSourceJob status\n",
						zap.Uint("jobId", id),
						zap.String("status", string(api.DescribeSourceJobCompletedWithFailure)),
						zap.Error(err),
					)
				}

				continue
			}

			// If the rest is SUCCEEDED, job has completed with no failure
			if status[api.DescribeResourceJobSucceeded] > 0 {
				err := s.db.UpdateDescribeSourceJob(id, api.DescribeSourceJobCompleted)
				if err != nil {
					s.logger.Error("Failed to update DescribeSourceJob status\n",
						zap.Uint("jobId", id),
						zap.String("status", string(api.DescribeSourceJobCompleted)),
						zap.Error(err),
					)
				}

				continue
			}
		}
	}
}

func (s Scheduler) RunDescribeJobScheduler() {
	s.logger.Info("Scheduling describe jobs on a timer")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for ; ; <-t.C {
		s.scheduleDescribeJob()
	}
}

func (s Scheduler) scheduleDescribeJob() {
	sources, err := s.db.QuerySourcesDueForDescribe()
	if err != nil {
		s.logger.Error("Failed to find the next sources to create DescribeSourceJob", zap.Error(err))
		DescribeJobsCount.WithLabelValues("failure").Inc()
		return
	}

	for _, source := range sources {
		if isPublishingBlocked(s.logger, s.describeJobQueue) {
			s.logger.Warn("The jobs in queue is over the threshold", zap.Error(err))
			return
		}

		s.logger.Info("Source is due for a describe. Creating a job now", zap.String("sourceId", source.ID.String()))
		daj := newDescribeSourceJob(source)
		err := s.db.CreateDescribeSourceJob(&daj)
		if err != nil {
			DescribeSourceJobsCount.WithLabelValues("failure").Inc()
			s.logger.Error("Failed to create DescribeSourceJob",
				zap.Uint("jobId", daj.ID),
				zap.String("sourceId", source.ID.String()),
				zap.Error(err),
			)
			continue
		}

		describedAt := time.Now()
		enqueueDescribeResourceJobs(s.logger, s.db, s.describeJobQueue, source, daj, describedAt)

		isSuccessful := true

		err = s.db.UpdateDescribeSourceJob(daj.ID, api.DescribeSourceJobInProgress)
		if err != nil {
			DescribeSourceJobsCount.WithLabelValues("failure").Inc()
			s.logger.Error("Failed to update DescribeSourceJob",
				zap.Uint("jobId", daj.ID),
				zap.String("sourceId", source.ID.String()),
				zap.Error(err),
			)
			isSuccessful = false
		}
		daj.Status = api.DescribeSourceJobInProgress

		err = s.db.UpdateSourceDescribed(source.ID, describedAt)
		if err != nil {
			DescribeSourceJobsCount.WithLabelValues("failure").Inc()
			s.logger.Error("Failed to update Source",
				zap.String("sourceId", source.ID.String()),
				zap.Error(err),
			)
			isSuccessful = false
		}

		if isSuccessful {
			DescribeSourceJobsCount.WithLabelValues("successful").Inc()
		}
	}
	DescribeJobsCount.WithLabelValues("successful").Inc()
}
func (s *Scheduler) RunComplianceReportCleanupJobScheduler() {
	s.logger.Info("Running compliance report cleanup job scheduler")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for range t.C {
		s.cleanupComplianceReportJob()
	}
}

func (s *Scheduler) RunDescribeCleanupJobScheduler() {
	s.logger.Info("Running describe cleanup job scheduler")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			s.cleanupDescribeJob()
		case sourceId := <-s.deletedSources:
			s.cleanupDescribeJobForSource(sourceId)
		}
	}
}

func (s Scheduler) cleanupDescribeJobForSource(sourceId string) {
	jobs, err := s.db.QueryDescribeSourceJobs(sourceId)
	if err != nil {
		s.logger.Error("Failed to find all completed DescribeSourceJob for source",
			zap.String("sourceId", sourceId),
			zap.Error(err),
		)
		DescribeCleanupJobsCount.WithLabelValues("failure").Inc()
		return
	}

	s.handleDescribeJobs(jobs)

	DescribeCleanupJobsCount.WithLabelValues("successful").Inc()
}

func (s Scheduler) handleDescribeJobs(jobs []DescribeSourceJob) {
	for _, sj := range jobs {
		// I purposefully didn't embbed this query in the previous query to keep returned results count low.
		drj, err := s.db.ListDescribeResourceJobs(sj.ID)
		if err != nil {
			s.logger.Error("Failed to retrieve DescribeResourceJobs for DescribeSouceJob",
				zap.Uint("jobId", sj.ID),
				zap.Error(err),
			)
			DescribeCleanupSourceJobsCount.WithLabelValues("failure").Inc()
			continue
		}

		success := true
		for _, rj := range drj {
			if isPublishingBlocked(s.logger, s.describeCleanupJobQueue) {
				s.logger.Warn("The jobs in queue is over the threshold")
				return
			}

			if err := s.describeCleanupJobQueue.Publish(DescribeCleanupJob{
				JobID:        rj.ID,
				ResourceType: rj.ResourceType,
			}); err != nil {
				s.logger.Error("Failed to publish describe clean up job to queue for DescribeResourceJob",
					zap.Uint("jobId", rj.ID),
					zap.Error(err),
				)
				success = false
				DescribeCleanupSourceJobsCount.WithLabelValues("failure").Inc()
				continue
			}

			err = s.db.DeleteDescribeResourceJob(rj.ID)
			if err != nil {
				s.logger.Error("Failed to delete DescribeResourceJob",
					zap.Uint("jobId", rj.ID),
					zap.Error(err),
				)
				success = false
				DescribeCleanupSourceJobsCount.WithLabelValues("failure").Inc()
				continue
			}
		}

		if success {
			err := s.db.DeleteDescribeSourceJob(sj.ID)
			if err != nil {
				s.logger.Error("Failed to delete DescribeSourceJob",
					zap.Uint("jobId", sj.ID),
					zap.Error(err),
				)
				DescribeCleanupSourceJobsCount.WithLabelValues("failure").Inc()
			} else {
				DescribeCleanupSourceJobsCount.WithLabelValues("successful").Inc()
			}
		} else {
			DescribeCleanupSourceJobsCount.WithLabelValues("failure").Inc()
		}

		s.logger.Info("Successfully deleted DescribeSourceJob and its DescribeResourceJobs",
			zap.Uint("jobId", sj.ID),
		)
	}
}

func (s Scheduler) cleanupDescribeJob() {
	jobs, err := s.db.QueryOlderThanNRecentCompletedDescribeSourceJobs(5)
	if err != nil {
		s.logger.Error("Failed to find older than 5 recent completed DescribeSourceJob for each source",
			zap.Error(err),
		)
		DescribeCleanupJobsCount.WithLabelValues("failure").Inc()
		return
	}

	s.handleDescribeJobs(jobs)

	DescribeCleanupJobsCount.WithLabelValues("successful").Inc()
}

func (s Scheduler) cleanupComplianceReportJob() {
	dsj, err := s.db.QueryOlderThanNRecentCompletedComplianceReportJobs(5)
	if err != nil {
		s.logger.Error("Failed to find older than 5 recent completed ComplianceReportJobs for each source",
			zap.Error(err),
		)

		return
	}

	for _, sj := range dsj {
		if err := s.complianceReportCleanupJobQueue.Publish(compliancereport.ComplianceReportCleanupJob{
			JobID: sj.ID,
		}); err != nil {
			s.logger.Error("Failed to publish compliance report clean up job to queue for ComplianceReportJob",
				zap.Uint("jobId", sj.ID),
				zap.Error(err),
			)
			return
		}

		err = s.db.DeleteComplianceReportJob(sj.ID)
		if err != nil {
			s.logger.Error("Failed to delete ComplianceReportJob",
				zap.Uint("jobId", sj.ID),
				zap.Error(err),
			)
		}

		s.logger.Info("Successfully deleted ComplianceReportJob",
			zap.Uint("jobId", sj.ID),
		)
	}
}

// Consume events from the source queue. Based on the action of the event,
// update the list of sources that need to be described. Either create a source
// or update/delete the source.
func (s *Scheduler) RunSourceEventsConsumer() error {
	s.logger.Info("Consuming messages from SourceEvents queue")
	msgs, err := s.sourceQueue.Consume()
	if err != nil {
		return err
	}

	for msg := range msgs {
		var event SourceEvent
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			s.logger.Error("Failed to unmarshal SourceEvent", zap.Error(err))
			err = msg.Nack(false, false)
			if err != nil {
				s.logger.Error("Failed nacking message", zap.Error(err))
			}
			continue
		}

		err := ProcessSourceAction(s.db, event)
		if err != nil {
			s.logger.Error("Failed to process event for Source",
				zap.String("sourceId", event.SourceID.String()),
				zap.Error(err),
			)
			err = msg.Nack(false, false)
			if err != nil {
				s.logger.Error("Failed nacking message", zap.Error(err))
			}
			continue
		}

		if err := msg.Ack(false); err != nil {
			s.logger.Error("Failed acking message", zap.Error(err))
		}

		if event.Action == SourceDelete {
			s.deletedSources <- event.SourceID.String()
		}
	}

	return fmt.Errorf("source events queue channel is closed")
}

// RunDescribeJobResultsConsumer consumes messages from the jobResult queue.
// It will update the status of the jobs in the database based on the message.
// It will also update the jobs status that are not completed in certain time to FAILED
func (s *Scheduler) RunDescribeJobResultsConsumer() error {
	s.logger.Info("Consuming messages from the JobResults queue")

	msgs, err := s.describeJobResultQueue.Consume()
	if err != nil {
		return err
	}

	t := time.NewTicker(JobTimeoutCheckInterval)
	defer t.Stop()

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("tasks channel is closed")
			}

			var result DescribeJobResult
			if err := json.Unmarshal(msg.Body, &result); err != nil {
				s.logger.Error("Failed to unmarshal DescribeResourceJob results\n", zap.Error(err))
				err = msg.Nack(false, false)
				if err != nil {
					s.logger.Error("Failed nacking message", zap.Error(err))
				}
				continue
			}

			s.logger.Info("Processing JobResult for Job",
				zap.Uint("jobId", result.JobID),
				zap.String("status", string(result.Status)),
			)
			err := s.db.UpdateDescribeResourceJobStatus(result.JobID, result.Status, result.Error)
			if err != nil {
				s.logger.Error("Failed to update the status of DescribeResourceJob",
					zap.Uint("jobId", result.JobID),
					zap.Error(err),
				)
				err = msg.Nack(false, true)
				if err != nil {
					s.logger.Error("Failed nacking message", zap.Error(err))
				}
				continue
			}

			if err := msg.Ack(false); err != nil {
				s.logger.Error("Failed acking message", zap.Error(err))
			}
		case <-t.C:
			err := s.db.UpdateDescribeResourceJobsTimedOut()
			if err != nil {
				s.logger.Error("Failed to update timed out DescribeResourceJobs", zap.Error(err))
			}
		}
	}
}

func (s *Scheduler) RunComplianceReportScheduler() {
	s.logger.Info("Scheduling ComplianceReport jobs on a timer")
	t := time.NewTicker(JobComplianceReportInterval)
	defer t.Stop()

	for ; ; <-t.C {
		sources, err := s.db.QuerySourcesDueForComplianceReport()
		if err != nil {
			s.logger.Error("Failed to find the next sources to create ComplianceReportJob", zap.Error(err))
			ComplianceJobsCount.WithLabelValues("failure").Inc()
			continue
		}

		for _, source := range sources {
			if isPublishingBlocked(s.logger, s.complianceReportJobQueue) {
				s.logger.Warn("The jobs in queue is over the threshold", zap.Error(err))
				break
			}

			s.logger.Error("Source is due for a steampipe check. Creating a ComplianceReportJob now", zap.String("sourceId", source.ID.String()))
			crj := newComplianceReportJob(source)
			err := s.db.CreateComplianceReportJob(&crj)
			if err != nil {
				ComplianceSourceJobsCount.WithLabelValues("failure").Inc()
				s.logger.Error("Failed to create ComplianceReportJob for Source",
					zap.Uint("jobId", crj.ID),
					zap.String("sourceId", source.ID.String()),
					zap.Error(err),
				)
				continue
			}

			enqueueComplianceReportJobs(s.logger, s.db, s.complianceReportJobQueue, source, &crj)

			err = s.db.UpdateSourceReportGenerated(source.ID)
			if err != nil {
				s.logger.Error("Failed to update report job of Source: %s\n", zap.String("sourceId", source.ID.String()), zap.Error(err))
				ComplianceSourceJobsCount.WithLabelValues("failure").Inc()
				continue
			}
			ComplianceSourceJobsCount.WithLabelValues("successful").Inc()
		}
		ComplianceJobsCount.WithLabelValues("successful").Inc()
	}
}

// RunComplianceReportJobResultsConsumer consumes messages from the complianceReportJobResultQueue queue.
// It will update the status of the jobs in the database based on the message.
// It will also update the jobs status that are not completed in certain time to FAILED
func (s *Scheduler) RunComplianceReportJobResultsConsumer() error {
	s.logger.Info("Consuming messages from the ComplianceReportJobResultQueue queue")

	msgs, err := s.complianceReportJobResultQueue.Consume()
	if err != nil {
		return err
	}

	t := time.NewTicker(JobTimeoutCheckInterval)
	defer t.Stop()

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("tasks channel is closed")
			}

			var result compliancereport.JobResult
			if err := json.Unmarshal(msg.Body, &result); err != nil {
				s.logger.Error("Failed to unmarshal ComplianceReportJob results", zap.Error(err))
				err = msg.Nack(false, false)
				if err != nil {
					s.logger.Error("Failed nacking message", zap.Error(err))
				}
				continue
			}

			s.logger.Info("Processing ReportJobResult for Job",
				zap.Uint("jobId", result.JobID),
				zap.String("status", string(result.Status)),
			)
			err := s.db.UpdateComplianceReportJob(result.JobID, result.Status, result.ReportCreatedAt, result.Error)
			if err != nil {
				s.logger.Error("Failed to update the status of ComplianceReportJob",
					zap.Uint("jobId", result.JobID),
					zap.Error(err))
				err = msg.Nack(false, true)
				if err != nil {
					s.logger.Error("Failed nacking message", zap.Error(err))
				}
				continue
			}

			if err := msg.Ack(false); err != nil {
				s.logger.Error("Failed acking message", zap.Error(err))
			}
		case <-t.C:
			err := s.db.UpdateComplianceReportJobsTimedOut()
			if err != nil {
				s.logger.Error("Failed to update timed out ComplianceReportJob", zap.Error(err))
			}
		}
	}
}

func (s *Scheduler) Stop() {
	queues := []queue.Interface{
		s.describeJobQueue,
		s.describeJobResultQueue,
		s.describeCleanupJobQueue,
		s.complianceReportJobQueue,
		s.complianceReportJobResultQueue,
		s.sourceQueue,
	}

	for _, queue := range queues {
		queue.Close()
	}
}

func newDescribeSourceJob(a Source) DescribeSourceJob {
	daj := DescribeSourceJob{
		SourceID:             a.ID,
		AccountID:            a.AccountID,
		DescribeResourceJobs: []DescribeResourceJob{},
		Status:               api.DescribeSourceJobCreated,
	}

	switch sType := api.SourceType(a.Type); sType {
	case api.SourceCloudAWS:
		for _, rType := range aws.ListResourceTypes() {
			daj.DescribeResourceJobs = append(daj.DescribeResourceJobs, DescribeResourceJob{
				ResourceType: rType,
				Status:       api.DescribeResourceJobCreated,
			})
		}
	case api.SourceCloudAzure:
		for _, rType := range azure.ListResourceTypes() {
			daj.DescribeResourceJobs = append(daj.DescribeResourceJobs, DescribeResourceJob{
				ResourceType: rType,
				Status:       api.DescribeResourceJobCreated,
			})
		}
	default:
		panic(fmt.Errorf("unsupported source type: %s", sType))
	}

	return daj
}

func newComplianceReportJob(a Source) ComplianceReportJob {
	return ComplianceReportJob{
		SourceID: a.ID,
		Status:   complianceapi.ComplianceReportJobCreated,
	}
}

func enqueueDescribeResourceJobs(logger *zap.Logger, db Database, q queue.Interface, a Source, daj DescribeSourceJob, describedAt time.Time) {
	for i, drj := range daj.DescribeResourceJobs {
		nextStatus := api.DescribeResourceJobQueued
		errMsg := ""

		if err := q.Publish(DescribeJob{
			JobID:        drj.ID,
			ParentJobID:  daj.ID,
			SourceID:     daj.SourceID.String(),
			AccountID:    daj.AccountID,
			SourceType:   a.Type,
			ResourceType: drj.ResourceType,
			DescribedAt:  describedAt.UnixMilli(),
			ConfigReg:    a.ConfigRef,
		}); err != nil {
			logger.Error("Failed to queue DescribeResourceJob",
				zap.Uint("jobId", drj.ID),
				zap.Error(err),
			)

			nextStatus = api.DescribeResourceJobFailed
			errMsg = fmt.Sprintf("queue: %s", err.Error())
		}

		if err := db.UpdateDescribeResourceJobStatus(drj.ID, nextStatus, errMsg); err != nil {
			logger.Error("Failed to update DescribeResourceJob",
				zap.Uint("jobId", drj.ID),
				zap.Error(err),
			)
		}

		daj.DescribeResourceJobs[i].Status = nextStatus
	}
}

func enqueueComplianceReportJobs(logger *zap.Logger, db Database, q queue.Interface, a Source, crj *ComplianceReportJob) {
	nextStatus := complianceapi.ComplianceReportJobInProgress
	errMsg := ""

	if err := q.Publish(compliancereport.Job{
		JobID:       crj.ID,
		SourceID:    a.ID,
		SourceType:  source.Type(a.Type),
		DescribedAt: a.LastDescribedAt.Time.UnixMilli(),
		ConfigReg:   a.ConfigRef,
		ReportID:    a.NextComplianceReportID,
	}); err != nil {
		logger.Error("Failed to queue ComplianceReportJob",
			zap.Uint("jobId", crj.ID),
			zap.Error(err),
		)

		nextStatus = complianceapi.ComplianceReportJobCompletedWithFailure
		errMsg = fmt.Sprintf("queue: %s", err.Error())
	}

	if err := db.UpdateComplianceReportJob(crj.ID, nextStatus, 0, errMsg); err != nil {
		logger.Error("Failed to update ComplianceReportJob",
			zap.Uint("jobId", crj.ID),
			zap.Error(err),
		)
	}

	crj.Status = nextStatus
}

func isPublishingBlocked(logger *zap.Logger, queue queue.Interface) bool {
	count, err := queue.Len()
	if err != nil {
		logger.Error("Failed to get queue len", zap.String("queueName", queue.Name()), zap.Error(err))
		DescribePublishingBlocked.WithLabelValues(queue.Name()).Set(0)
		return false
	}
	if count >= MaxJobInQueue {
		DescribePublishingBlocked.WithLabelValues(queue.Name()).Set(1)
		return true
	}
	DescribePublishingBlocked.WithLabelValues(queue.Name()).Set(0)
	return false
}
