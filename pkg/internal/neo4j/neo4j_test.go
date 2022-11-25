package neo4j

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

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
	ctx := context.Background()

	pool, err := dockertest.NewPool("")
	require.NoError(t, err, "connect to docker")

	net, err := pool.CreateNetwork("keibi-neo4j")
	require.NoError(t, err, "create network")
	t.Cleanup(func() {
		require.NoError(t, pool.RemoveNetwork(net), "remove network")
	})

	rootPassword := "654321"
	user, pass, port := "inventory", "123456", "7687"
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Name:         "keibi-neo4j",
		Repository:   "neo4j",
		Tag:          "5.1.0-community",
		ExposedPorts: []string{port},
		Env: []string{
			"NEO4J_AUTH=neo4j/" + rootPassword,
		},
		Networks: []*dockertest.Network{net},
	})
	require.NoError(t, err, "status neo4j")
	t.Cleanup(func() {
		require.NoError(t, pool.Purge(resource), "purge resource")
	})
	time.Sleep(time.Second * 10)

	// create user
	exitCode, err := resource.Exec([]string{"cypher-shell", "-u", "neo4j", "-p", rootPassword, "CREATE USER " + user + " SET PLAINTEXT PASSWORD '" + pass + "' SET PASSWORD CHANGE NOT REQUIRED;"}, dockertest.ExecOptions{})
	time.Sleep(time.Second * 5)
	require.NoError(t, err, "create user")
	require.Equal(t, 0, exitCode, "create user")

	cfg := &Config{
		Host:   GetDockerHost(),
		Port:   resource.GetPort("7687/tcp"),
		User:   user,
		Passwd: pass,
	}
	logger, err := zap.NewProduction()
	require.NoError(t, err, "new zap logger")

	driver, err := NewDriver(cfg, logger)
	require.NoError(t, err, "new driver")

	session := driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeWrite,
	})
	_, err = session.Run(ctx, "CREATE (n:Person {name: $name, title: $title}), (m:Person {name: $name2, title: $title2}), (n)-[:KNOWS]->(m)", map[string]interface{}{
		"name":   "Arthur",
		"title":  "King",
		"name2":  "Merlin",
		"title2": "Wizard",
	})
	require.NoError(t, err, "run query")
}
