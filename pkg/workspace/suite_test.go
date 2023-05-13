package workspace

import (
	idocker "github.com/kaytu-io/kaytu-util/pkg/dockertest"
	"testing"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/google/uuid"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSuite(t *testing.T) {
	suite.Run(t, &testSuite{})
}

type testSuite struct {
	suite.Suite

	server *Server

	name         string
	owner        uuid.UUID
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
	time.Sleep(time.Second * 5)

	logger, err := zap.NewProduction()
	s.NoError(err, "new zap logger")
	cfg := &Config{
		Host:         idocker.GetDockerHost(),
		Port:         resource.GetPort("5432/tcp"),
		User:         user,
		Password:     pass,
		DBName:       name,
		DomainSuffix: ".app.keibi.io",
	}
	db, err := NewDatabase(cfg, logger)
	s.NoError(err, "new database")

	s.server = &Server{
		db:  db,
		cfg: cfg,
	}

	scheme := runtime.NewScheme()
	s.NoError(helmv2.AddToScheme(scheme), "add scheme")
	s.NoError(corev1.AddToScheme(scheme), "add scheme")
	s.server.kubeClient = fake.NewClientBuilder().WithScheme(scheme).Build()

	s.name = "geeks"
	s.owner = uuid.New()
	s.domainSuffix = cfg.DomainSuffix
}

func (ts *testSuite) TearDownSuite() {
}

func (ts *testSuite) TearDownTest() {
	tx := ts.server.db.orm.Exec("delete from workspaces")
	ts.NoError(tx.Error)
}
