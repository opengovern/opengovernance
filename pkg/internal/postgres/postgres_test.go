package postgres

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Workspace struct {
	gorm.Model

	ID          uuid.UUID `json:"id"`
	Name        string    `gorm:"uniqueIndex" json:"name"`
	OwnerId     uuid.UUID `json:"owner_id"`
	Domain      string    `json:"domain"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
}

func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}
	return value
}

func GetDockerHost() string {
	return getEnv("DOCKERTEST_HOST", "localhost")
}

func TestNewClient(t *testing.T) {
	pool, err := dockertest.NewPool("")
	require.NoError(t, err, "connect to docker")

	net, err := pool.CreateNetwork("keibi")
	require.NoError(t, err, "create network")
	t.Cleanup(func() {
		require.NoError(t, pool.RemoveNetwork(net), "remove network")
	})

	user, pass, name, port := "postgres", "123456", "workspace", "5432"
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Name:         "keibi",
		Repository:   "postgres",
		Tag:          "12.2-alpine",
		ExposedPorts: []string{port},
		Env: []string{
			"POSTGRES_USER=" + user,
			"POSTGRES_PASSWORD=" + pass,
			"POSTGRES_DB=" + name,
		},
		Networks: []*dockertest.Network{net},
	})
	require.NoError(t, err, "status postgres")
	t.Cleanup(func() {
		require.NoError(t, pool.Purge(resource), "purge resource")
	})
	time.Sleep(time.Second * 5)

	cfg := &Config{
		Host:   GetDockerHost(),
		Port:   resource.GetPort("5432/tcp"),
		User:   user,
		Passwd: pass,
		DB:     name,
	}
	logger, err := zap.NewProduction()
	require.NoError(t, err, "new zap logger")

	orm, err := NewClient(cfg, logger)
	require.NoError(t, err, "new client")

	orm.AutoMigrate(&Workspace{})
	require.NoError(t, err, "auto migrate")
}
