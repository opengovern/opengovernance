package steampipe

import (
	"context"
	"database/sql"
	"testing"

	"github.com/jackc/pgx/v4"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
	idocker "gitlab.com/keibiengine/keibi-engine/pkg/internal/dockertest"
)

func TestNewStampipeDatabase(t *testing.T) {
	pool, err := dockertest.NewPool("")
	require.NoError(t, err, "connect to docker")

	net, err := pool.CreateNetwork("keibi")
	require.NoError(t, err)
	t.Cleanup(func() {
		err = pool.RemoveNetwork(net)
		require.NoError(t, err, "remove network")
	})

	user, pass, name, port := "postgres", "123456", "streampipe", "5432"
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Name:         "keibi_test_psql",
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
		err := pool.Purge(resource)
		require.NoError(t, err, "purge resource %s", resource)
	})

	option := Option{
		Host: idocker.GetDockerHost(),
		Port: resource.GetPort("5432/tcp"),
		User: user,
		Pass: pass,
		Db:   name,
	}

	var db *Database
	err = pool.Retry(func() error {
		db, err = NewSteampipeDatabase(option)
		if err != nil {
			return err
		}
		return nil
	})
	require.NoError(t, err, "wait for postgres connection")
	defer db.conn.Close()

	ctx := context.Background()
	tx, err := db.conn.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)

	var count sql.NullInt64
	err = db.conn.QueryRow(ctx, "SELECT count(*) from information_schema.tables").Scan(&count)
	require.NoError(t, err)
	require.Greater(t, count.Int64, int64(0))
	require.NoError(t, tx.Commit(ctx))
}
