package steampipe

import (
	"github.com/kaytu-io/kaytu-util/pkg/koanf"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
)

func New(config koanf.Postgres) (*steampipe.Database, error) {
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

	return db, nil
}
