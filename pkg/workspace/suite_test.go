package workspace

import (
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/suite"
	idocker "gitlab.com/keibiengine/keibi-engine/pkg/internal/dockertest"
)

func TestSuite(t *testing.T) {
	suite.Run(t, &testSuite{})
}

type testSuite struct {
	suite.Suite

	db *Database
}

func (s *testSuite) SetupSuite() {
	t := s.T()

	pool, err := dockertest.NewPool("")
	s.NoError(err, "connect to docker")

	net, err := pool.CreateNetwork("keibi")
	s.NoError(err, "create network")
	t.Cleanup(func() {
		s.NoError(pool.RemoveNetwork(net), "remove network")
	})

	user, pass, name, port := "postgres", "123456", "test", "5432"
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Name:         "keibi_workspace",
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
	s.NoError(err, "status postgres")
	t.Cleanup(func() {
		s.NoError(pool.Purge(resource), "purge resource")
	})
	time.Sleep(time.Second * 3)

	settings := Config{
		Host:     idocker.GetDockerHost(),
		Port:     resource.GetPort("5432/tcp"),
		User:     user,
		Password: pass,
		DBName:   name,
	}
	db, err := NewDatabase(&settings)
	s.NoError(err, "new database")
	s.db = db
}

func (ts *testSuite) TearDownSuite() {
}

func (ts *testSuite) TearDownTest() {
	// tx := ts.handler.db.orm.Exec("delete from benchmarks where id = ?", ts.benchmarkId)
	// ts.NoError(tx.Error)
	// tx = ts.handler.db.orm.Exec("delete from benchmark_assignments where benchmark_id = ?", ts.benchmarkId)
	// ts.NoError(tx.Error)
}
