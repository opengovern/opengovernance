package onboard

import (
	"fmt"
	"testing"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestCreateSource(t *testing.T) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("Could not connect to docker: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.Run("postgres", "latest", []string{"POSTGRES_PASSWORD=mysecretpassword"})
	if err != nil {
		t.Fatalf("Could not start resource: %s", err)
	}

	var db *gorm.DB
	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := pool.Retry(func() error {
		db, err = gorm.Open(postgres.Open(fmt.Sprintf("postgres://postgres:mysecretpassword@localhost:%s/postgres", resource.GetPort("5432/tcp"))), &gorm.Config{})
		if err != nil {
			return err
		}
		if err != nil {
			return err
		}

		d, _ := db.DB()
		return d.Ping()
	}); err != nil {
		t.Fatalf("Could not connect to database: %s", err)
	}

	// Enable uuid_generate_v4
	db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`)

	db.AutoMigrate(
		&Organization{},
		&Source{},
		&AWSMetadata{},
	)

	s := Source{}
	d := Database{db}
	_, err = d.CreateSource(&s)
	assert.NoError(t, err)

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		t.Fatalf("Could not purge resource: %s", err)
	}
}
