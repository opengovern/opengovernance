package describe

import (
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/insight"
	insightapi "gitlab.com/keibiengine/keibi-engine/pkg/insight/api"

	api2 "gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"

	"github.com/cenkalti/backoff/v3"
	compliance_report "gitlab.com/keibiengine/keibi-engine/pkg/compliance"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/queue"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/dockertest"
	mocksqueue "gitlab.com/keibiengine/keibi-engine/pkg/internal/queue/mocks"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SchedulerTestSuite struct {
	suite.Suite

	orm      *gorm.DB
	rabbit   dockertest.RabbitMQServer
	queueInt queue.Interface
	Scheduler
}

func (s *SchedulerTestSuite) SetupSuite() {
	s.orm = dockertest.StartupPostgreSQL(s.T())
	s.rabbit = dockertest.StartupRabbitMQ(s.T())
}

func (s *SchedulerTestSuite) BeforeTest(suiteName, testName string) {
	require := s.Require()

	logger, err := zap.NewDevelopment()
	require.NoError(err, "logger")

	cfg := queue.Config{
		Server: s.rabbit,
	}
	cfg.Queue.Name = ComplianceReportJobsQueueName
	cfg.Queue.Durable = true
	cfg.Consumer.ID = "test-scheduler"
	queueInt, err := queue.New(cfg)
	s.Require().NoError(err, "queue")

	s.queueInt = queueInt
	db := Database{
		orm: s.orm,
	}
	s.Scheduler = Scheduler{
		id: "test-scheduler",
		db: db,
		//describeJobQueue:                &mocksqueue.Interface{},
		//describeJobResultQueue:          &mocksqueue.Interface{},
		describeCleanupJobQueue:         &mocksqueue.Interface{},
		complianceReportJobQueue:        &mocksqueue.Interface{},
		complianceReportJobResultQueue:  &mocksqueue.Interface{},
		complianceReportCleanupJobQueue: &mocksqueue.Interface{},
		insightJobQueue:                 &mocksqueue.Interface{},
		insightJobResultQueue:           &mocksqueue.Interface{},
		logger:                          logger,
		deletedSources:                  make(chan string, ConcurrentDeletedSources),
	}
	s.Scheduler.httpServer = NewHTTPServer("localhost:2345", db, &s.Scheduler)

	err = s.Scheduler.db.Initialize()
	require.NoError(err, "initialize db")
}

func (s *SchedulerTestSuite) AfterTest(suiteName, testName string) {
	require := s.Require()

	tx := s.Scheduler.db.orm.Exec("DROP TABLE IF EXISTS describe_resource_jobs;")
	require.NoError(tx.Error, "drop describe_resource_jobs")

	tx = s.Scheduler.db.orm.Exec("DROP TABLE IF EXISTS describe_source_jobs;")
	require.NoError(tx.Error, "drop describe_source_jobs")

	tx = s.Scheduler.db.orm.Exec("DROP TABLE IF EXISTS compliance_report_jobs;")
	require.NoError(tx.Error, "drop compliance_report_jobs")

	tx = s.Scheduler.db.orm.Exec("DROP TABLE IF EXISTS sources;")
	require.NoError(tx.Error, "drop sources")

	s.queueInt.Close()
	s.Scheduler = Scheduler{}
}

func (s *SchedulerTestSuite) TestSourceEventCreate() {
	require := s.Require()

	uuid := uuid.New()
	err := ProcessSourceAction(s.Scheduler.db, SourceEvent{
		Action:     SourceCreate,
		SourceID:   uuid,
		AccountID:  "1234567890",
		SourceType: api.SourceCloudAWS,
		ConfigRef:  "config/ref/path",
	})
	require.NoError(err, "create source")

	db := s.Scheduler.db

	sources, err := db.ListSources()
	require.NoError(err, "list sources")
	require.Equal(1, len(sources))
	require.Equal(uuid, sources[0].ID, uuid)
	require.Equal(api.SourceCloudAWS, sources[0].Type)
	require.Equal("config/ref/path", sources[0].ConfigRef)
	require.False(sources[0].LastDescribedAt.Valid)
	require.True(sources[0].NextDescribeAt.Valid)
}

func (s *SchedulerTestSuite) TestSourceEventUpdate() {
	require := s.Require()

	uuid := uuid.New()
	err := ProcessSourceAction(s.Scheduler.db, SourceEvent{
		Action:     SourceCreate,
		SourceID:   uuid,
		AccountID:  "1234567890",
		SourceType: api.SourceCloudAWS,
		ConfigRef:  "config/ref/path",
	})
	require.NoError(err, "create source")

	err = ProcessSourceAction(s.Scheduler.db, SourceEvent{
		Action:     SourceUpdate,
		SourceID:   uuid,
		AccountID:  "1234567890",
		SourceType: api.SourceCloudAzure,
		ConfigRef:  "config/ref/path2",
	})
	require.NoError(err, "update source")

	db := s.Scheduler.db

	sources, err := db.ListSources()
	require.NoError(err, "list sources")
	require.Equal(1, len(sources))
	require.Equal(uuid, sources[0].ID, uuid)
	require.Equal(api.SourceCloudAzure, sources[0].Type)
	require.Equal("config/ref/path2", sources[0].ConfigRef)
	require.False(sources[0].LastDescribedAt.Valid)
	require.True(sources[0].NextDescribeAt.Valid)
}

func (s *SchedulerTestSuite) TestSourceEventDelete() {
	require := s.Require()

	uuid := uuid.New()
	err := ProcessSourceAction(s.Scheduler.db, SourceEvent{
		Action:     SourceCreate,
		SourceID:   uuid,
		AccountID:  "1234567890",
		SourceType: api.SourceCloudAWS,
		ConfigRef:  "config/ref/path",
	})
	require.NoError(err, "create source")

	db := s.Scheduler.db

	sources, err := db.ListSources()
	require.NoError(err, "list sources")
	require.Equal(1, len(sources))

	err = ProcessSourceAction(s.Scheduler.db, SourceEvent{
		Action:   SourceDelete,
		SourceID: uuid,
	})
	require.NoError(err, "delete source")

	sources, err = db.ListSources()
	require.NoError(err, "list sources")
	require.Equal(0, len(sources))
}

func (s *SchedulerTestSuite) TestDescribeJobQueue_NoSource() {
	require := s.Require()
	db := s.Scheduler.db

	s.scheduleDescribeJob() // Shouldn't have any error with no sources

	sources, err := db.ListSources()
	require.NoError(err, "list sources")
	require.Equal(0, len(sources))

	dsj, err := db.ListAllDescribeSourceJobs()
	require.NoError(err, "list all describe source jobs")
	require.Equal(0, len(dsj))

	drj, err := db.ListAllDescribeResourceJobs()
	require.NoError(err, "list all describe resource jobs")
	require.Equal(0, len(drj))
}

func (s *SchedulerTestSuite) TestDescribeJobQueue() {
	require := s.Require()
	db := s.Scheduler.db

	uuid := uuid.New()
	err := ProcessSourceAction(s.Scheduler.db, SourceEvent{
		Action:     SourceCreate,
		SourceID:   uuid,
		AccountID:  "1234567890",
		SourceType: api.SourceCloudAWS,
		ConfigRef:  "config/ref/path",
	})
	require.NoError(err, "create source")

	//s.Scheduler.describeJobQueue.(*mocksqueue.Interface).On("Publish", mock.Anything).Return(error(nil))
	//s.Scheduler.describeJobQueue.(*mocksqueue.Interface).On("Len", mock.Anything).Return(0, nil)
	//s.Scheduler.describeJobQueue.(*mocksqueue.Interface).On("Name", mock.Anything).Return("temp")

	s.scheduleDescribeJob()

	dsj, err := db.ListAllDescribeSourceJobs()
	require.NoError(err, "list all describe source jobs")
	require.Equal(1, len(dsj))

	drj, err := db.ListAllDescribeResourceJobs()
	require.NoError(err, "list all describe resource jobs")
	require.Equal(len(aws.ListResourceTypes()), len(drj))
}

func (s *SchedulerTestSuite) TestDescribeResultJobQueue() {
	require := s.Require()

	// Setup job results consumer
	ch := make(chan amqp.Delivery)
	defer close(ch)

	//s.Scheduler.describeJobResultQueue.(*mocksqueue.Interface).
	//	On("Consume").
	//	Return((<-chan amqp.Delivery)(ch), nil)

	go func() {
		err := s.RunDescribeJobResultsConsumer()
		require.Error(err, "tasks channel is closed")
	}()

	// Create a fake source with jobs
	source := Source{
		ID:   uuid.New(),
		Type: api.SourceCloudAWS,
		DescribeSourceJobs: []DescribeSourceJob{
			{
				DescribeResourceJobs: []DescribeResourceJob{
					{
						ResourceType: "resource-type-test",
					},
				},
			},
		},
	}

	err := s.db.CreateSource(&source)
	require.NoError(err, "create source")

	body, err := json.Marshal(DescribeJobResult{
		JobID:       source.DescribeSourceJobs[0].DescribeResourceJobs[0].ID,
		ParentJobID: source.DescribeSourceJobs[0].ID,
		Status:      api.DescribeResourceJobSucceeded,
	})
	require.NoError(err, "marshal job result")

	// Insert a delivery object into the result queue
	ack := mocksqueue.Acknowledger{}
	ack.On("Ack", uint64(1), false).Return(error(nil))
	ch <- amqp.Delivery{
		Acknowledger: &ack,
		Body:         body,
		DeliveryTag:  1,
	}

	// The job status is completed
	done := false
	for !done {
		select {
		case <-time.After(10 * time.Millisecond):
			job, err := s.db.GetDescribeResourceJob(source.DescribeSourceJobs[0].DescribeResourceJobs[0].ID)
			if err != nil {
				require.NoError(err, "get resource job")
			}

			if job.Status == api.DescribeResourceJobSucceeded {
				require.Equal("", job.FailureMessage)
				done = true
			}
		case <-time.After(10 * time.Second):
			require.FailNow("timed out: job result not updated properly")
		}
	}

	//s.Scheduler.describeJobResultQueue.(*mocksqueue.Interface).AssertCalled(s.T(), "Consume")
	ack.AssertCalled(s.T(), "Ack", uint64(1), false)
	ack.AssertNumberOfCalls(s.T(), "Ack", 1)
}

func (s *SchedulerTestSuite) TestUpdateNextDescribeAt() {
	require := s.Require()

	id := uuid.New()
	err := s.db.CreateSource(&Source{
		ID:                     id,
		Type:                   api.SourceCloudAWS,
		ConfigRef:              "aws/config",
		LastDescribedAt:        sql.NullTime{},
		NextDescribeAt:         sql.NullTime{},
		LastComplianceReportAt: sql.NullTime{},
		NextComplianceReportAt: sql.NullTime{},
		DescribeSourceJobs:     nil,
		ComplianceReportJobs:   nil,
	})
	require.NoError(err)

	err = s.db.UpdateSourceDescribed(id, time.Now(), 2*time.Hour)
	require.NoError(err)

	source, err := s.db.GetSourceByUUID(id)
	require.NoError(err)
	require.True(source.LastDescribedAt.Valid)
	require.True(source.LastDescribedAt.Time.Before(time.Now()))
	require.True(source.NextDescribeAt.Valid)
	require.True(source.NextDescribeAt.Time.After(time.Now().Add(time.Hour)))
}

func (s *SchedulerTestSuite) TestDescribeResultJobQueueFailed() {
	require := s.Require()

	// Setup job results consumer
	ch := make(chan amqp.Delivery)
	defer close(ch)

	//s.Scheduler.describeJobResultQueue.(*mocksqueue.Interface).
	//	On("Consume").
	//	Return((<-chan amqp.Delivery)(ch), nil)

	go func() {
		err := s.RunDescribeJobResultsConsumer()
		require.Error(err, "tasks channel is closed")
	}()

	// Create a fake source with jobs
	source := Source{
		ID:   uuid.New(),
		Type: api.SourceCloudAWS,
		DescribeSourceJobs: []DescribeSourceJob{
			{
				DescribeResourceJobs: []DescribeResourceJob{
					{
						ResourceType: "resource-type-test",
					},
				},
			},
		},
	}

	err := s.db.CreateSource(&source)
	require.NoError(err, "create source")

	body, err := json.Marshal(DescribeJobResult{
		JobID:       source.DescribeSourceJobs[0].DescribeResourceJobs[0].ID,
		ParentJobID: source.DescribeSourceJobs[0].ID,
		Status:      api.DescribeResourceJobFailed,
		Error:       "failed to describe",
	})
	require.NoError(err, "marshal job result")

	// Insert a delivery object into the result queue
	ack := mocksqueue.Acknowledger{}
	ack.On("Ack", uint64(1), false).Return(error(nil))
	ch <- amqp.Delivery{
		Acknowledger: &ack,
		Body:         body,
		DeliveryTag:  1,
	}

	// The job status is completed
	done := false
	for !done {
		select {
		case <-time.After(10 * time.Millisecond):
			job, err := s.db.GetDescribeResourceJob(source.DescribeSourceJobs[0].DescribeResourceJobs[0].ID)
			if err != nil {
				require.NoError(err, "get resource job")
			}

			if job.Status == api.DescribeResourceJobFailed {
				require.Equal("failed to describe", job.FailureMessage)
				done = true
			}
		case <-time.After(10 * time.Second):
			require.FailNow("timed out: job result not updated properly")
		}
	}

	//s.Scheduler.describeJobResultQueue.(*mocksqueue.Interface).AssertCalled(s.T(), "Consume")
	ack.AssertCalled(s.T(), "Ack", uint64(1), false)
	ack.AssertNumberOfCalls(s.T(), "Ack", 1)
}

func (s *SchedulerTestSuite) TestDescribeResultJobQueueInvalid() {
	require := s.Require()

	// Setup job results consumer
	ch := make(chan amqp.Delivery)
	defer close(ch)

	//s.Scheduler.describeJobResultQueue.(*mocksqueue.Interface).
	//	On("Consume").
	//	Return((<-chan amqp.Delivery)(ch), nil)

	go func() {
		err := s.RunDescribeJobResultsConsumer()
		require.Error(err, "tasks channel is closed")
	}()

	// Insert an invalid delivery object into the result queue
	ack := mocksqueue.Acknowledger{}
	ack.On("Nack", uint64(1), false, false).Return(error(nil))
	ch <- amqp.Delivery{
		Acknowledger: &ack,
		Body:         []byte(`{"invalid_json":}`),
		DeliveryTag:  1,
	}

	<-time.After(10 * time.Millisecond)
	//s.Scheduler.describeJobResultQueue.(*mocksqueue.Interface).AssertCalled(s.T(), "Consume")
	ack.AssertCalled(s.T(), "Nack", uint64(1), false, false)
}

func (s *SchedulerTestSuite) TestDescribeCleanup_NoJob() {
	s.cleanupDescribeJob() // Shouldn't fail
	s.Scheduler.describeCleanupJobQueue.(*mocksqueue.Interface).AssertNotCalled(s.T(), "Publish", mock.Anything)
}

func (s *SchedulerTestSuite) TestDescribeCleanup_NothingToClean() {
	require := s.Require()

	// Create a fake source with jobs
	source := Source{
		ID:   uuid.New(),
		Type: api.SourceCloudAWS,
		DescribeSourceJobs: []DescribeSourceJob{
			{
				Model: gorm.Model{
					ID: 1,
				},
				Status: api.DescribeSourceJobCompleted,
			},
			{
				Model: gorm.Model{
					ID: 2,
				},
				Status: api.DescribeSourceJobCompleted,
			},
			{
				Model: gorm.Model{
					ID: 3,
				},
				Status: api.DescribeSourceJobCompleted,
			},
			{
				Model: gorm.Model{
					ID: 4,
				},
				Status: api.DescribeSourceJobCompleted,
			},
			{
				Model: gorm.Model{
					ID: 5,
				},
				Status: api.DescribeSourceJobCompleted,
			},
		},
	}

	err := s.db.CreateSource(&source)
	require.NoError(err, "create source")

	s.cleanupDescribeJob()

	// Nothing should be deleted
	s.Scheduler.describeCleanupJobQueue.(*mocksqueue.Interface).AssertNotCalled(s.T(), "Publish", mock.Anything)

	sources, err := s.db.ListDescribeSourceJobs(source.ID)
	require.NoError(err, "list describe source jobs")
	require.Equal(5, len(sources))
}

func (s *SchedulerTestSuite) TestDescribeCleanup_NothingReadyToClean() {
	require := s.Require()

	// Create a fake source with jobs
	// There are more jobs than 5 but they are not all completed
	source := Source{
		ID:   uuid.New(),
		Type: api.SourceCloudAWS,
		DescribeSourceJobs: []DescribeSourceJob{
			{
				Model: gorm.Model{
					ID: 1,
				},
				Status: api.DescribeSourceJobCompleted,
			},
			{
				Model: gorm.Model{
					ID: 2,
				},
				Status: api.DescribeSourceJobCompleted,
			},
			{
				Model: gorm.Model{
					ID: 3,
				},
				Status: api.DescribeSourceJobCompleted,
			},
			{
				Model: gorm.Model{
					ID: 4,
				},
				Status: api.DescribeSourceJobCompleted,
			},
			{
				Model: gorm.Model{
					ID: 5,
				},
				Status: api.DescribeSourceJobCreated,
			},
			{
				Model: gorm.Model{
					ID: 6,
				},
				Status: api.DescribeSourceJobInProgress,
			},
		},
	}

	err := s.db.CreateSource(&source)
	require.NoError(err, "create source")

	s.cleanupDescribeJob()

	// Nothing should be deleted
	s.Scheduler.describeCleanupJobQueue.(*mocksqueue.Interface).AssertNotCalled(s.T(), "Publish", mock.Anything)

	sources, err := s.db.ListDescribeSourceJobs(source.ID)
	require.NoError(err, "list describe source jobs")
	require.Equal(6, len(sources))
}

func (s *SchedulerTestSuite) TestDescribeCleanup_DeleteSource() {
	require := s.Require()

	id := uuid.New()
	sourceJobs, err := s.db.ListDescribeSourceJobs(id)
	require.NoError(err, "list describe source jobs")
	s.Equal(0, len(sourceJobs))

	// Create a fake source with jobs
	source := Source{
		ID:   id,
		Type: api.SourceCloudAWS,
		DescribeSourceJobs: []DescribeSourceJob{
			{
				Model: gorm.Model{
					ID: 1,
				},
				Status: api.DescribeSourceJobCompleted,
			},
			{
				Model: gorm.Model{
					ID: 2,
				},
				Status: api.DescribeSourceJobCompleted,
			},
			{
				Model: gorm.Model{
					ID: 3,
				},
				Status: api.DescribeSourceJobCompleted,
			},
			{
				Model: gorm.Model{
					ID: 4,
				},
				Status: api.DescribeSourceJobCompleted,
			},
			{
				Model: gorm.Model{
					ID: 5,
				},
				Status: api.DescribeSourceJobCompleted,
			},
		},
		ComplianceReportJobs: []ComplianceReportJob{
			{
				Model: gorm.Model{
					ID: 1,
				},
				Status: api2.ComplianceReportJobCompleted,
			},
			{
				Model: gorm.Model{
					ID: 2,
				},
				Status: api2.ComplianceReportJobCompleted,
			},
			{
				Model: gorm.Model{
					ID: 3,
				},
				Status: api2.ComplianceReportJobCompleted,
			},
			{
				Model: gorm.Model{
					ID: 4,
				},
				Status: api2.ComplianceReportJobCompleted,
			},
			{
				Model: gorm.Model{
					ID: 5,
				},
				Status: api2.ComplianceReportJobCompleted,
			},
		},
	}

	err = s.db.CreateSource(&source)
	require.NoError(err, "create source")

	sourceJobs, err = s.db.ListDescribeSourceJobs(source.ID)
	require.NoError(err, "list describe source jobs")
	s.Equal(len(source.DescribeSourceJobs), len(sourceJobs))

	reportJobs, err := s.db.ListComplianceReportJobs(source.ID)
	require.NoError(err, "list compliance report jobs")
	s.Equal(len(source.ComplianceReportJobs), len(reportJobs))

	s.cleanupDescribeJobForDeletedSource(id.String())
}

func (s *SchedulerTestSuite) TestDescribeCleanup() {
	require := s.Require()

	s.Scheduler.describeCleanupJobQueue.(*mocksqueue.Interface).On("Publish", mock.Anything).Return(error(nil))

	// Create a fake source with jobs
	// There are more jobs than 5 but they are not all completed
	t := time.Now()
	source := Source{
		ID:   uuid.New(),
		Type: api.SourceCloudAWS,
		DescribeSourceJobs: []DescribeSourceJob{
			{
				Model: gorm.Model{
					ID:        1,
					UpdatedAt: t.Add(time.Second),
				},
				Status: api.DescribeSourceJobCompleted,
				DescribeResourceJobs: []DescribeResourceJob{
					{
						Model: gorm.Model{
							ID: 1,
						},
						ResourceType: "resource-type-1",
					},
					{
						Model: gorm.Model{
							ID: 2,
						},
						ResourceType: "resource-type-2",
					},
				},
			},
			{
				Model: gorm.Model{
					ID:        2,
					UpdatedAt: t.Add(2 * time.Second),
				},
				Status: api.DescribeSourceJobCompleted,
			},
			{
				Model: gorm.Model{
					ID:        3,
					UpdatedAt: t.Add(3 * time.Second),
				},
				Status: api.DescribeSourceJobCompleted,
			},
			{
				Model: gorm.Model{
					ID:        4,
					UpdatedAt: t.Add(4 * time.Second),
				},
				Status: api.DescribeSourceJobCompleted,
			},
			{
				Model: gorm.Model{
					ID:        5,
					UpdatedAt: t.Add(5 * time.Second),
				},
				Status: api.DescribeSourceJobCompletedWithFailure,
			},
			{
				Model: gorm.Model{
					ID:        6,
					UpdatedAt: t.Add(6 * time.Second),
				},
				Status: api.DescribeSourceJobCompleted,
			},
		},
	}

	err := s.db.CreateSource(&source)
	require.NoError(err, "create source")

	s.Scheduler.describeCleanupJobQueue.(*mocksqueue.Interface).On("Len", mock.Anything).Return(0, nil)
	s.Scheduler.describeCleanupJobQueue.(*mocksqueue.Interface).On("Name", mock.Anything).Return("temp")

	s.cleanupDescribeJob()

	// Cleanup should be called
	s.Scheduler.describeCleanupJobQueue.(*mocksqueue.Interface).
		AssertCalled(s.T(), "Publish", DescribeCleanupJob{
			JobID:        1,
			ResourceType: "resource-type-1",
		})
	s.Scheduler.describeCleanupJobQueue.(*mocksqueue.Interface).
		AssertCalled(s.T(), "Publish", DescribeCleanupJob{
			JobID:        2,
			ResourceType: "resource-type-2",
		})

	sources, err := s.db.ListDescribeSourceJobs(source.ID)
	require.NoError(err, "list describe source jobs")
	require.Equal(5, len(sources))

	for _, dsj := range sources {
		require.NotEqual(1, dsj.ID, "job 1 should've been deleted")
	}
}

func (s *SchedulerTestSuite) TestRunComplianceReport() {
	source := Source{
		ID:   uuid.New(),
		Type: api.SourceCloudAWS,
		NextComplianceReportAt: sql.NullTime{
			Time:  time.Now().Add(-60 * time.Second),
			Valid: true,
		},
	}
	err := s.Scheduler.db.CreateSource(&source)
	s.Require().NoError(err)

	s.Require().NotNil(s.Scheduler)
	s.Require().NotNil(s.Scheduler.complianceReportJobQueue)

	delivery, err := s.Scheduler.complianceReportJobQueue.Consume()
	s.Require().NoError(err)

	//go s.Scheduler.RunComplianceReportScheduler()

	err = backoff.Retry(func() error {
		jobs, err := s.Scheduler.db.ListComplianceReportJobs(source.ID)
		if err != nil {
			return err
		}

		if jobs == nil || len(jobs) < 1 {
			return errors.New("job not found")
		}

		for _, job := range jobs {
			if job.SourceID == source.ID {
				if job.Status != api2.ComplianceReportJobInProgress {
					return errors.New("job not in progress")
				}
			}
		}

		sources, err := s.Scheduler.db.ListSources()
		if err != nil {
			return err
		}

		for _, src := range sources {
			if src.ID == source.ID {
				v, err := src.NextComplianceReportAt.Value()
				if err != nil {
					return err
				}

				t := v.(time.Time)
				if t.Before(time.Now()) {
					return errors.New("time hasn't updated")
				}
			}
		}

		select {
		case msg := <-delivery:
			var job compliance_report.Job
			err := json.Unmarshal(msg.Body, &job)
			if err != nil {
				return err
			}

			return nil
		default:
			return errors.New("msg not received")
		}
	}, backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Second), 30))
	s.Require().NoError(err)
}

func (s *SchedulerTestSuite) TestRunInsightJob() {
	s.Scheduler.insightJobQueue.(*mocksqueue.Interface).On("Len", mock.Anything).Return(0, nil)
	s.Scheduler.insightJobQueue.(*mocksqueue.Interface).On("Name", mock.Anything).Return("temp")
	s.Scheduler.insightJobQueue.(*mocksqueue.Interface).On("Publish", mock.Anything).Return(error(nil))

	ins := Insight{
		Description: "this is a test insight",
		Query:       "select count(*) from aws_ec2_instance",
		Connector:   "AWS",
		Category:    "IAM",
	}

	err := s.Scheduler.db.AddInsight(&ins)
	s.Require().NoError(err)

	go s.Scheduler.RunInsightJobScheduler()

	err = backoff.Retry(func() error {
		jobs, err := s.Scheduler.db.ListInsightJobs()
		if err != nil {
			return err
		}

		if jobs == nil || len(jobs) < 1 {
			return errors.New("job not found")
		}

		for _, job := range jobs {
			if job.InsightID == ins.ID {
				if job.Status != insightapi.InsightJobInProgress {
					return errors.New("job not in progress")
				}
			}
		}

		insJob, err := s.Scheduler.db.FetchLastInsightJob()
		if err != nil {
			return err
		}
		s.Require().True(insJob.CreatedAt.Add(5 * time.Minute).After(time.Now()))

		return nil
	}, backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Second), 30))
	s.Require().NoError(err)

	isOK := false
	for _, call := range s.Scheduler.insightJobQueue.(*mocksqueue.Interface).Calls {
		if call.Method == "Publish" {
			if v, ok := call.Arguments.Get(0).(insight.Job); ok {
				if v.QueryID == ins.ID {
					isOK = true
				}
			}
		}
	}
	s.Require().True(isOK)
}

func TestScheduler(t *testing.T) {
	suite.Run(t, &SchedulerTestSuite{})
}
