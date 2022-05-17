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

	server *Server

	name         string
	owner        string
	domainSuffix string
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

	user, pass, name, port := "postgres", "123456", "workspace", "5432"
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
	time.Sleep(time.Second * 2)

	cfg := &Config{
		Host:         idocker.GetDockerHost(),
		Port:         resource.GetPort("5432/tcp"),
		User:         user,
		Password:     pass,
		DBName:       name,
		DomainSuffix: ".app.keibi.io",
	}
	db, err := NewDatabase(cfg)
	s.NoError(err, "new database")

	s.server = &Server{
		db:  db,
		cfg: cfg,
	}
	s.name = "geeksforgeeks"
	s.owner = "00000000-0000-0000-0000-000000000000"
	s.domainSuffix = cfg.DomainSuffix
}

func (ts *testSuite) TearDownSuite() {
}

func (ts *testSuite) TearDownTest() {
	tx := ts.server.db.db.Exec("delete from workspaces")
	ts.NoError(tx.Error)
}
