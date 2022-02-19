package onboard

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupDB(t *testing.T) *gorm.DB {
	dsn, ok := os.LookupEnv("DB_URL")
	if !ok {
		t.Fatal("DB_URL must exist")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	db.AutoMigrate(
		&Organization{},
		&Source{},
		&AWSMetadata{},
	)

	return db
}

func TestGetSource(t *testing.T) {
	db := setupDB(t)
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
	db := setupDB(t)
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
