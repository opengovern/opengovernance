package onboard

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestGetSource(t *testing.T) {
	dsn := "postgres://postgres:mysecretpassword@localhost:5432/postgres"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	db.AutoMigrate(
		&Organization{},
		&Source{},
		&AWSMetadata{},
		&AzureMetadata{},
	)
 
	r := InitializeRouter()
	h := &HttpHandler{db: Database{db}}
	h.Register(r.Group("/api/v1"))

	orgId := uuid.New()
	db.Create(&Organization{
		ID:          orgId,
		Name:        "123123123",
		Description: "123123123",
		AdminEmail:  "me@example.com",
		KeibiUrl:    "12312312312313213",
		CreatedAt:   time.Now().UTC(),
	})

	srcId := uuid.New()
	db.Create(&Source{
		ID:             srcId,
		SourceId:       "12312312312312321",
		OrganizationID: orgId,
		Name:           "123123",
		Description:    "123123123",
		Type:           SourceCloudAWS,
		CreatedAt:      time.Now().UTC(),
	})

	req := httptest.NewRequest(echo.GET, fmt.Sprintf("/api/v1/organizations/%s/sources/%s", orgId, srcId), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetOrganization(t *testing.T) {
	dsn := "postgres://postgres:mysecretpassword@localhost:5432/postgres"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	db.AutoMigrate(
		&Organization{},
		&Source{},
		&AWSMetadata{},
		&AzureMetadata{},
	)
 
	r := InitializeRouter()
	h := &HttpHandler{db: Database{db}}
	h.Register(r.Group("/api/v1"))

	orgId := uuid.New()
	db.Create(&Organization{
		ID:          orgId,
		Name:        "123123123",
		Description: "123123123",
		AdminEmail:  "me@example.com",
		KeibiUrl:    "12312312312313213",
		CreatedAt:   time.Now().UTC(),
	})

	req := httptest.NewRequest(echo.GET, fmt.Sprintf("/api/v1/organizations/%s", orgId), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
