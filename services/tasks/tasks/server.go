package tasks

import (
	"github.com/microsoft/durabletask-go/backend"
	"github.com/microsoft/durabletask-go/backend/postgres"
	"github.com/microsoft/durabletask-go/task"
	"github.com/opengovern/og-util/pkg/koanf"
	"golang.org/x/net/context"
	"strconv"
)

func CreateScheduler(cfg koanf.Postgres) error {
	ctx := context.Background()

	r := task.NewTaskRegistry()
	err := r.AddOrchestrator(ActivitySequenceOrchestrator)
	if err != nil {
		return err
	}
	err = r.AddActivity(SayHelloActivity)
	if err != nil {
		return err
	}

	client, worker, err := Init(ctx, r, cfg)
	if err != nil {
		return err
	}

}

// Init creates and initializes an in-memory client and worker pair with default configuration.
func Init(ctx context.Context, r *task.TaskRegistry, cfg koanf.Postgres) (backend.TaskHubClient, backend.TaskHubWorker, error) {
	logger := backend.DefaultLogger()

	// Create an executor
	executor := task.NewTaskExecutor(r)

	port, err := strconv.ParseUint(cfg.Port, 10, 16)
	if err != nil {
		return nil, nil, err
	}

	// Create a new backend
	// Use the in-memory sqlite provider by specifying ""
	options := postgres.NewPostgresOptions(cfg.Host, uint16(port), cfg.DB, cfg.Username, cfg.Password)
	be := postgres.NewPostgresBackend(options, backend.DefaultLogger())
	orchestrationWorker := backend.NewOrchestrationWorker(be, executor, logger)
	activityWorker := backend.NewActivityTaskWorker(be, executor, logger)
	taskHubWorker := backend.NewTaskHubWorker(be, orchestrationWorker, activityWorker, logger)

	// Start the worker
	err = taskHubWorker.Start(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Get the client to the backend
	taskHubClient := backend.NewTaskHubClient(be)

	return taskHubClient, taskHubWorker, nil
}
