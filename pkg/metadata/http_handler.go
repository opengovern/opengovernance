package metadata

import (
	"fmt"
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/opengovern/og-util/pkg/postgres"
	"github.com/opengovern/opengovernance/pkg/metadata/config"
	"github.com/opengovern/opengovernance/pkg/metadata/internal/database"
	db2 "github.com/opengovern/opengovernance/services/migrator/db"
	"github.com/opengovern/opengovernance/services/migrator/db/model"
	"go.uber.org/zap"
	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HttpHandler struct {
	cfg        config.Config
	db         database.Database
	migratorDb *db2.Database
	kubeClient client.Client
	logger     *zap.Logger
}

func InitializeHttpHandler(
	cfg config.Config,
	logger *zap.Logger,
) (*HttpHandler, error) {

	fmt.Println("Initializing http handler")

	psqlCfg := postgres.Config{
		Host:    cfg.Postgres.Host,
		Port:    cfg.Postgres.Port,
		User:    cfg.Postgres.Username,
		Passwd:  cfg.Postgres.Password,
		DB:      cfg.Postgres.DB,
		SSLMode: cfg.Postgres.SSLMode,
	}
	orm, err := postgres.NewClient(&psqlCfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new postgres client: %w", err)
	}
	logger.Info("Connected to the postgres database", zap.String("database", cfg.Postgres.DB))

	db := database.NewDatabase(orm)
	err = db.Initialize()
	if err != nil {
		return nil, err
	}
	logger.Info("Initialized database", zap.String("database", cfg.Postgres.DB))

	migratorDbCfg := postgres.Config{
		Host:    cfg.Postgres.Host,
		Port:    cfg.Postgres.Port,
		User:    cfg.Postgres.Username,
		Passwd:  cfg.Postgres.Password,
		DB:      "migrator",
		SSLMode: cfg.Postgres.SSLMode,
	}
	migratorOrm, err := postgres.NewClient(&migratorDbCfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new postgres client: %w", err)
	}
	if err := migratorOrm.AutoMigrate(&model.Migration{}); err != nil {
		return nil, fmt.Errorf("gorm migrate: %w", err)
	}
	migratorDb := &db2.Database{ORM: migratorOrm}

	kubeClient, err := NewKubeClient()
	if err != nil {
		return nil, err
	}
	err = v1.AddToScheme(kubeClient.Scheme())
	if err != nil {
		return nil, fmt.Errorf("add v1 to scheme: %w", err)
	}

	return &HttpHandler{
		cfg:        cfg,
		db:         db,
		migratorDb: migratorDb,
		kubeClient: kubeClient,
		logger:     logger,
	}, nil
}

func NewKubeClient() (client.Client, error) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := helmv2.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := v1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	kubeClient, err := client.New(ctrl.GetConfigOrDie(), client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	return kubeClient, nil
}
