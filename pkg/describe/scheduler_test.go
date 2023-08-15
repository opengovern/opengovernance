package describe

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/dockertest"
	"github.com/kaytu-io/kaytu-util/pkg/queue"
	"github.com/kaytu-io/kaytu-util/pkg/queue/mocks"

	api2 "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"

	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
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
		complianceReportJobQueue:        &mocks.Interface{},
		complianceReportJobResultQueue:  &mocks.Interface{},
		complianceReportCleanupJobQueue: &mocks.Interface{},
		insightJobQueue:                 &mocks.Interface{},
		insightJobResultQueue:           &mocks.Interface{},
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
	ack := mocks.Acknowledger{}
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
	ack := mocks.Acknowledger{}
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
	ack := mocks.Acknowledger{}
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
	s.Scheduler.describeCleanupJobQueue.(*mocks.Interface).AssertNotCalled(s.T(), "Publish", mock.Anything)
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
	s.Scheduler.describeCleanupJobQueue.(*mocks.Interface).AssertNotCalled(s.T(), "Publish", mock.Anything)

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
	s.Scheduler.describeCleanupJobQueue.(*mocks.Interface).AssertNotCalled(s.T(), "Publish", mock.Anything)

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

	s.Scheduler.describeCleanupJobQueue.(*mocks.Interface).On("Publish", mock.Anything).Return(error(nil))

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

	s.Scheduler.describeCleanupJobQueue.(*mocks.Interface).On("Len", mock.Anything).Return(0, nil)
	s.Scheduler.describeCleanupJobQueue.(*mocks.Interface).On("Name", mock.Anything).Return("temp")

	s.cleanupDescribeJob()

	// Cleanup should be called
	s.Scheduler.describeCleanupJobQueue.(*mocks.Interface).
		AssertCalled(s.T(), "Publish", DescribeCleanupJob{
			JobType:      DescribeCleanupJobTypeInclusiveDelete,
			JobIDs:       []uint{1},
			ResourceType: "resource-type-1",
		})
	s.Scheduler.describeCleanupJobQueue.(*mocks.Interface).
		AssertCalled(s.T(), "Publish", DescribeCleanupJob{
			JobType:      DescribeCleanupJobTypeInclusiveDelete,
			JobIDs:       []uint{2},
			ResourceType: "resource-type-2",
		})

	sources, err := s.db.ListDescribeSourceJobs(source.ID)
	require.NoError(err, "list describe source jobs")
	require.Equal(5, len(sources))

	for _, dsj := range sources {
		require.NotEqual(1, dsj.ID, "job 1 should've been deleted")
	}
}

func TestScheduler(t *testing.T) {
	suite.Run(t, &SchedulerTestSuite{})
}
