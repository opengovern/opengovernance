package steampipe

import (
	"github.com/kaytu-io/kaytu-util/pkg/koanf"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"go.uber.org/zap"
)

func New(config koanf.Postgres, logger *zap.Logger) (*steampipe.Database, error) {
	logger = logger.Named("streampipe")

	db, err := steampipe.NewSteampipeDatabase(steampipe.Option{
		Host: config.Host,
		Port: config.Port,
		User: config.Username,
		Pass: config.Password,
		Db:   config.DB,
	})
	if err != nil {
		return nil, err
	}

	logger.Info("Successfully connected to the steampipe database", zap.String("database", config.DB))

	return db, nil
}
