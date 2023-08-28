package reporter

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/kaytu-io/kaytu-util/pkg/queue"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/config"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"go.uber.org/zap"
)

//go:embed queries-aws.json
var awsQueriesStr string
var awsQueries []Query

//go:embed queries-azure.json
var azureQueriesStr string
var azureQueries []Query

type Query struct {
	ListQuery string   `json:"list"`
	GetQuery  string   `json:"get"`
	KeyFields []string `json:"keyFields"`
	TableName string   `json:"tableName"`
}

type TriggerQueryRequest struct {
	Queries []Query `json:"queries"`
	Source  string  `json:"source"`
}

type QueryMismatch struct {
	KeyColumn      string `json:"keyColumn"`
	ConflictColumn string `json:"conflictColumn"`
	Steampipe      string `json:"steampipe"`
	Elasticsearch  string `json:"elasticsearch"`
}

type TriggerQueryResponse struct {
	TotalRows          int             `json:"totalRows"`
	Query              Query           `json:"query"`
	NotMatchingColumns []string        `json:"notMatchingColumns"`
	Mismatches         []QueryMismatch `json:"messages"`
}

type ServiceConfig struct {
	Database        config.Postgres
	RabbitMQ        config.RabbitMQ
	Steampipe       config.Postgres
	SteampipeES     config.Postgres
	Onboard         config.KaytuService
	ScheduleMinutes int
}

type Service struct {
	steampipe       *steampipe.Database
	esSteampipe     *steampipe.Database
	db              *Database
	jobsQueue       queue.Interface
	onboardClient   onboardClient.OnboardServiceClient
	logger          *zap.Logger
	ScheduleMinutes int
}

type Job struct {
	ID           uint    `json:"id"`
	ConnectionId string  `json:"connectionId"`
	Queries      []Query `json:"queries"`
}

var ReporterJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "kaytu",
	Subsystem: "reporter",
	Name:      "job_total",
	Help:      "Count of reporter jobs",
}, []string{"table_name", "status"})

func New(config ServiceConfig, logger *zap.Logger) (*Service, error) {
	if content, err := os.ReadFile("/queries-aws.json"); err == nil {
		awsQueriesStr = string(content)
	}
	if content, err := os.ReadFile("/queries-azure.json"); err == nil {
		azureQueriesStr = string(content)
	}

	if err := json.Unmarshal([]byte(awsQueriesStr), &awsQueries); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(azureQueriesStr), &azureQueries); err != nil {
		return nil, err
	}

	s2, err := steampipe.NewSteampipeDatabase(steampipe.Option{
		Host: config.SteampipeES.Host,
		Port: config.SteampipeES.Port,
		User: config.SteampipeES.Username,
		Pass: config.SteampipeES.Password,
		Db:   config.SteampipeES.DB,
	})
	if err != nil {
		return nil, err
	}

	cfg := postgres.Config{
		Host:    config.Database.Host,
		Port:    config.Database.Port,
		User:    config.Database.Username,
		Passwd:  config.Database.Password,
		DB:      config.Database.DB,
		SSLMode: config.Database.SSLMode,
	}
	orm, err := postgres.NewClient(&cfg, logger)
	if err != nil {
		return nil, err
	}
	db, err := NewDatabase(orm)
	if err != nil {
		return nil, err
	}

	qCfg := queue.Config{}
	qCfg.Server.Username = config.RabbitMQ.Username
	qCfg.Server.Password = config.RabbitMQ.Password
	qCfg.Server.Host = config.RabbitMQ.Service
	qCfg.Server.Port = 5672
	qCfg.Queue.Name = ReporterQueueName
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = "reporter-service"
	reporterQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	onboard := onboardClient.NewOnboardServiceClient(config.Onboard.BaseURL, nil)

	if config.ScheduleMinutes <= 0 {
		config.ScheduleMinutes = 5
	}

	return &Service{
		steampipe:       nil,
		esSteampipe:     s2,
		db:              db,
		jobsQueue:       reporterQueue,
		onboardClient:   onboard,
		logger:          logger,
		ScheduleMinutes: config.ScheduleMinutes,
	}, nil
}

func (s *Service) Run() {
	fmt.Println("starting scheduling")
	for _, q := range awsQueries {
		s.logger.Info("loaded aws query ", zap.String("listQuery", q.ListQuery))
	}
	for _, q := range azureQueries {
		s.logger.Info("loaded azure query ", zap.String("listQuery", q.ListQuery))
	}

	for {
		//fmt.Println("starting job")
		if err := s.TriggerRandomJob(); err != nil {
			s.logger.Error("failed to run job", zap.Error(err))
			time.Sleep(time.Minute)
		} else {
			time.Sleep(time.Duration(s.ScheduleMinutes) * time.Minute)
		}
	}
}

func (s *Service) TriggerRandomJob() error {
	account, err := s.RandomAccount()
	if err != nil {
		return err
	}

	query := s.RandomQuery(account.Connector)
	if query != nil {
		_, err = s.TriggerJob(account.ID.String(), []Query{*query})
		return err
	}
	return fmt.Errorf("no query found")
}

func (s *Service) TriggerJob(connectionId string, queries []Query) (*DatabaseWorkerJob, error) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("panic", zap.Error(fmt.Errorf("%v", r)))
		}
	}()

	dbJob := DatabaseWorkerJob{
		ConnectionID: connectionId,
		Status:       JobStatusPending,
	}
	err := s.db.InsertWorkerJob(&dbJob)
	if err != nil {
		s.logger.Error("failed to insert job", zap.Error(err))
		return nil, err
	}

	job := Job{
		ID:           dbJob.ID,
		ConnectionId: connectionId,
		Queries:      queries,
	}
	if err := s.jobsQueue.Publish(job); err != nil {
		s.logger.Error("failed to publish job", zap.Error(err))
		return nil, err
	}

	return &dbJob, nil
}

func (s *Service) RandomAccount() (*api2.Connection, error) {
	srcs, err := s.onboardClient.ListSources(&httpclient.Context{
		UserRole: api.AdminRole,
	}, nil)
	if err != nil {
		return nil, err
	}

	idx := rand.Intn(len(srcs))
	return &srcs[idx], nil
}

func (s *Service) RandomQuery(sourceType source.Type) *Query {
	switch sourceType {
	case source.CloudAWS:
		idx := rand.Intn(len(awsQueries))
		return &awsQueries[idx]
	case source.CloudAzure:
		idx := rand.Intn(len(azureQueries))
		return &azureQueries[idx]
	}
	return nil
}

func PopulateSteampipe(logger *zap.Logger, account *api2.Connection, awsCred *api2.AWSCredentialConfig, azureCred *api2.AzureCredentialConfig) error {
	dirname, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	filePath := path.Join(dirname, ".steampipe", "config", "steampipe.spc")
	os.MkdirAll(filepath.Dir(filePath), os.ModePerm)

	if awsCred != nil {
		assumeRoleConfigs := ""
		if awsCred.AssumeRoleName != "" && awsCred.AccountId != account.ConnectionID {
			logger.Info("assuming role", zap.String("role", awsCred.AssumeRoleName), zap.String("accountID", awsCred.AccountId))
			assumeRoleConfigs = fmt.Sprintf("role_arn = arn:aws:iam::%s:role/%s\n", account.ConnectionID, awsCred.AssumeRoleName)
			if awsCred.ExternalId != nil {
				assumeRoleConfigs += fmt.Sprintf("external_id = %s\n", *awsCred.ExternalId)
			}
		}

		credFilePath := path.Join(dirname, ".aws", "credentials")
		os.MkdirAll(filepath.Dir(credFilePath), os.ModePerm)
		content := fmt.Sprintf(`
[default]
aws_access_key_id = %s
aws_secret_access_key = %s

[reporter]
region = us-east-1
source_profile = default
%s
`,
			awsCred.AccessKey, awsCred.SecretKey, assumeRoleConfigs)
		err = os.WriteFile(credFilePath, []byte(content), os.ModePerm)
		if err != nil {
			return err
		}

		//os.Setenv("AWS_ACCESS_KEY_ID", awsCred.AccessKey)
		//os.Setenv("AWS_SECRET_ACCESS_KEY", awsCred.SecretKey)
		content = `
connection "aws" {
  plugin  = "aws"
  regions = ["*"]
  profile = "reporter"
}
`
		filePath = path.Join(dirname, ".steampipe", "config", "aws.spc")
		return os.WriteFile(filePath, []byte(content), os.ModePerm)
	}

	if azureCred != nil {
		content := fmt.Sprintf(`
connection "azure" {
  plugin = "azure"
  tenant_id       = "%s"
  subscription_id = "%s"
  client_id       = "%s"
  client_secret   = "%s"
}
`,
			azureCred.TenantId, account.ConnectionID, azureCred.ClientId, azureCred.ClientSecret)
		filePath = dirname + "/.steampipe/config/azure.spc"
		err = os.WriteFile(filePath, []byte(content), os.ModePerm)
		if err != nil {
			return err
		}

		content = fmt.Sprintf(`
connection "azuread" {
  plugin = "azuread"
  tenant_id       = "%s"
  client_id       = "%s"
  client_secret   = "%s"
}
`,
			azureCred.TenantId, azureCred.ClientId, azureCred.ClientSecret)
		filePath = dirname + "/.steampipe/config/azuread.spc"
		return os.WriteFile(filePath, []byte(content), os.ModePerm)
	}

	return nil
}

// json2 should be es and json1 should be steampipe
func compareJsons(j1, j2 []byte) bool {
	var o1 map[string]any
	err := json.Unmarshal(j1, &o1)
	if err != nil {
		return false
	}

	var o2 map[string]any
	err = json.Unmarshal(j2, &o2)
	if err != nil {
		return false
	}

	for k, v := range o1 {
		if v2, ok := o2[k]; ok {
			if v2 != v {
				return false
			}
		} else {
			return false
		}
	}
	return true
}

func (w *Worker) Do(j Job) ([]TriggerQueryResponse, error) {
	connection, err := w.onboardClient.GetSource(&httpclient.Context{
		UserRole: api.InternalRole,
	}, j.ConnectionId)
	if err != nil {
		w.logger.Error("failed to get source", zap.Error(err))
		return nil, err
	}
	w.logger.Info("got connection", zap.String("account", connection.ConnectionID))

	awsCred, azureCred, err := w.onboardClient.GetSourceFullCred(&httpclient.Context{
		UserRole: api.KaytuAdminRole,
	}, connection.ID.String())
	if err != nil {
		w.logger.Error("failed to get source full cred", zap.Error(err))
		return nil, err
	}
	err = PopulateSteampipe(w.logger, connection, awsCred, azureCred)
	if err != nil {
		w.logger.Error("failed to populate steampipe", zap.Error(err))
		return nil, err
	}
	stdOut, stdErr := exec.Command("steampipe", "plugin", "update", "--all").CombinedOutput()
	if stdErr != nil {
		w.logger.Error("failed to start steampipe", zap.Error(stdErr), zap.String("output", string(stdOut)))
		return nil, stdErr
	}
	w.logger.Info("steampipe plugins updated")

	stdOut, stdErr = exec.Command("steampipe", "service", "start", "--database-listen", "network", "--database-port",
		"9193", "--database-password", "abcd").CombinedOutput()
	if stdErr != nil {
		w.logger.Error("failed to start steampipe", zap.Error(stdErr), zap.String("output", string(stdOut)))
		return nil, stdErr
	}

	// Do not remove this, steampipe will not start without this
	homeDir, _ := os.UserHomeDir()
	stdOut, stdErr = exec.Command("rm", path.Join(homeDir, ".steampipe", "config", "default.spc")).CombinedOutput()
	if stdErr != nil {
		w.logger.Error("failed to remove default.spc", zap.Error(stdErr), zap.String("output", string(stdOut)))
		return nil, stdErr
	}

	w.logger.Info("steampipe started")

	stdOut, stdErr = exec.Command("steampipe", "plugin", "list").CombinedOutput()
	if stdErr != nil {
		w.logger.Error("failed to list steampipe plugins", zap.Error(err), zap.String("output", string(stdOut)))
		return nil, stdErr
	}
	w.logger.Info("steampipe plugins", zap.String("output", string(stdOut)))

	originalSteampipe, err := steampipe.NewSteampipeDatabase(steampipe.Option{
		Host: "localhost",
		Port: "9193",
		User: "steampipe",
		Pass: "abcd",
		Db:   "steampipe",
	})
	if err != nil {
		w.logger.Error("failed to connect to steampipe", zap.Error(err))
		return nil, err
	}
	defer originalSteampipe.Conn().Close()

	var results []TriggerQueryResponse
	for _, query := range j.Queries {
		w.logger.Info("running query", zap.String("account", connection.ConnectionID), zap.String("query", query.ListQuery))
		listQuery := strings.ReplaceAll(query.ListQuery, "%ACCOUNT_ID%", connection.ConnectionID)
		listQuery = strings.ReplaceAll(listQuery, "%KAYTU_ACCOUNT_ID%", connection.ID.String())
		steampipeRows, err := originalSteampipe.Conn().Query(context.Background(), listQuery)
		if err != nil {
			w.logger.Error("failed to run query", zap.Error(err), zap.String("query", query.ListQuery), zap.String("account", connection.ConnectionID))
			return nil, err
		}

		var mismatches []QueryMismatch
		var columns []string
		rowCount := 0
		for steampipeRows.Next() {
			rowCount++
			steampipeRow, err := steampipeRows.Values()
			if err != nil {
				w.logger.Error("failed to get steampipe row values",
					zap.Error(err),
					zap.String("query", query.ListQuery),
					zap.String("account", connection.ConnectionID),
					zap.Any("row", steampipeRow))
				return nil, err
			}

			steampipeRecord := map[string]any{}
			for idx, field := range steampipeRows.FieldDescriptions() {
				steampipeRecord[string(field.Name)] = steampipeRow[idx]
			}

			getQuery := strings.ReplaceAll(query.GetQuery, "%ACCOUNT_ID%", connection.ConnectionID)
			getQuery = strings.ReplaceAll(getQuery, "%KAYTU_ACCOUNT_ID%", connection.ID.String())

			var keyValues []any
			for _, f := range query.KeyFields {
				keyValues = append(keyValues, steampipeRecord[f])
			}

			esRows, err := w.kaytuSteampipeDb.Conn().Query(context.Background(), getQuery, keyValues...)
			if err != nil {
				w.logger.Error("failed to run query", zap.Error(err), zap.String("query", query.GetQuery), zap.String("account", connection.ConnectionID))
				return nil, err
			}

			found := false
			w.logger.Info("comparing steampipe and es records", zap.Int("number", rowCount))
			for esRows.Next() {
				esRow, err := esRows.Values()
				if err != nil {
					w.logger.Error("failed to get es row values",
						zap.Error(err),
						zap.String("query", query.GetQuery),
						zap.String("account", connection.ConnectionID),
						zap.Any("row", esRow))
					return nil, err
				}

				found = true

				esRecord := map[string]any{}
				for idx, field := range esRows.FieldDescriptions() {
					esRecord[string(field.Name)] = esRow[idx]
				}

				for k, v := range steampipeRecord {
					v2 := esRecord[k]

					j1, err := json.Marshal(v)
					if err != nil {
						return nil, err
					}

					j2, err := json.Marshal(v2)
					if err != nil {
						return nil, err
					}

					sj1 := strings.ToLower(string(j1))
					sj2 := strings.ToLower(string(j2))

					if sj1 == "null" {
						sj1 = "{}"
					}
					if sj2 == "null" {
						sj2 = "{}"
					}

					if sj1 != sj2 {
						if compareJsons(j2, j1) {
							ReporterJobsCount.WithLabelValues(query.TableName, "Succeeded").Inc()
							continue
						}
						ReporterJobsCount.WithLabelValues(query.TableName, "Failed").Inc()
						hasColumn := false
						for _, c := range columns {
							if c == k {
								hasColumn = true
								break
							}
						}
						if !hasColumn {
							columns = append(columns, k)
						}
						mismatches = append(mismatches, QueryMismatch{
							KeyColumn:      fmt.Sprintf("%v", keyValues),
							ConflictColumn: k,
							Steampipe:      sj1,
							Elasticsearch:  sj2,
						})
						if k != "etag" && k != "tags" {
							w.logger.Warn("inconsistency in data",
								zap.String("get-query", query.GetQuery),
								zap.String("accountID", connection.ConnectionID),
								zap.String("steampipe", sj1),
								zap.String("es", sj2),
								zap.String("conflictColumn", k),
								zap.String("keyColumns", fmt.Sprintf("%v", keyValues)),
							)
						}
					} else {
						ReporterJobsCount.WithLabelValues(query.TableName, "Succeeded").Inc()
					}
				}
			}
			esRows.Close()

			if !found {
				mismatches = append(mismatches, QueryMismatch{
					KeyColumn:      fmt.Sprintf("%v", keyValues),
					ConflictColumn: "",
					Steampipe:      "",
					Elasticsearch:  "Record Not Found",
				})
				ReporterJobsCount.WithLabelValues(query.TableName, "Failed").Inc()
				w.logger.Warn("record not found",
					zap.String("get-query", query.GetQuery),
					zap.String("accountID", connection.ConnectionID),
					zap.String("keyColumns", fmt.Sprintf("%v", keyValues)),
				)
			}
		}
		steampipeRows.Close()
		results = append(results, TriggerQueryResponse{
			TotalRows:          rowCount,
			Query:              query,
			NotMatchingColumns: columns,
			Mismatches:         mismatches,
		})
	}

	return results, nil
}
