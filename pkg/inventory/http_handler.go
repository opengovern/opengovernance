package inventory

import (
	"fmt"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type HttpHandler struct {
	client        keibi.Client
	db            Database
	steampipeConn *SteampipeDatabase
}

func InitializeHttpHandler(
	elasticSearchAddress string,
	elasticSearchUsername string,
	elasticSearchPassword string,
	postgresHost string,
	postgresPort string,
	postgresDb string,
	postgresUsername string,
	postgresPassword string,
	steampipeHost string,
	steampipePort string,
	steampipeDb string,
	steampipeUsername string,
	steampipePassword string,
) (h *HttpHandler, err error) {

	h = &HttpHandler{}

	fmt.Println("Initializing http handler")

	// setup postgres connection
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
	h.db = Database{orm: db}
	fmt.Println("Connected to the postgres database: ", postgresDb)

	err = h.db.Initialize()
	if err != nil {
		return nil, err
	}
	fmt.Println("Initialized postgres database: ", postgresDb)

	// setup steampipe connection
	h.steampipeConn, err = NewSteampipeDatabase(SteampipeOption{
		Host: steampipeHost,
		Port: steampipePort,
		User: steampipeUsername,
		Pass: steampipePassword,
		Db:   steampipeDb,
	})

	defaultAccountID := "default"
	h.client, err = keibi.NewClient(keibi.ClientConfig{
		Addresses: []string{elasticSearchAddress},
		Username:  &elasticSearchUsername,
		Password:  &elasticSearchPassword,
		AccountID: &defaultAccountID,
	})
	if err != nil {
		return nil, err
	}

	return h, nil
}
