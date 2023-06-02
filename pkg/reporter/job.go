package reporter

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/config"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	api2 "gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"
	onboardClient "gitlab.com/keibiengine/keibi-engine/pkg/onboard/client"
	"go.uber.org/zap"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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
}

type JobConfig struct {
	Steampipe       config.Postgres
	SteampipeES     config.Postgres
	Onboard         config.KeibiService
	ScheduleMinutes int
}

type Job struct {
	steampipe       *steampipe.Database
	esSteampipe     *steampipe.Database
	onboardClient   onboardClient.OnboardServiceClient
	logger          *zap.Logger
	ScheduleMinutes int
}

func New(config JobConfig) (*Job, error) {
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

	installCmd := exec.Command("steampipe", "plugin", "install", "steampipe")
	err := installCmd.Run()
	if err != nil {
		return nil, err
	}

	installCmd = exec.Command("steampipe", "plugin", "install", "aws")
	err = installCmd.Run()
	if err != nil {
		return nil, err
	}

	installCmd = exec.Command("steampipe", "plugin", "install", "azure")
	err = installCmd.Run()
	if err != nil {
		return nil, err
	}

	installCmd = exec.Command("steampipe", "plugin", "install", "azuread")
	err = installCmd.Run()
	if err != nil {
		return nil, err
	}

	cmdSteampipe := exec.Command("steampipe", "service", "start", "--database-listen", "network", "--database-port",
		"9193", "--database-password", "abcd")
	err = cmdSteampipe.Run()
	if err != nil {
		return nil, err
	}
	time.Sleep(5 * time.Second)
	fmt.Println("Steampipe service started")

	s1, err := steampipe.NewSteampipeDatabase(steampipe.Option{
		Host: "localhost",
		Port: "9193",
		User: "steampipe",
		Pass: "abcd",
		Db:   "steampipe",
	})
	if err != nil {
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

	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}

	onboard := onboardClient.NewOnboardServiceClient(config.Onboard.BaseURL, nil)

	if config.ScheduleMinutes <= 0 {
		config.ScheduleMinutes = 5
	}

	return &Job{
		steampipe:       s1,
		esSteampipe:     s2,
		onboardClient:   onboard,
		logger:          logger,
		ScheduleMinutes: config.ScheduleMinutes,
	}, nil
}

func (j *Job) Run() {
	fmt.Println("starting scheduling")
	for _, q := range awsQueries {
		j.logger.Info("loaded aws query ", zap.String("listQuery", q.ListQuery))
	}
	for _, q := range azureQueries {
		j.logger.Info("loaded azure query ", zap.String("listQuery", q.ListQuery))
	}

	for {
		//fmt.Println("starting job")
		if err := j.RunJob(); err != nil {
			j.logger.Error("failed to run job", zap.Error(err))
		}
		time.Sleep(time.Duration(j.ScheduleMinutes) * time.Minute)
	}
}

func (j *Job) RunJob() error {
	defer func() {
		if r := recover(); r != nil {
			j.logger.Error("panic", zap.Error(fmt.Errorf("%v", r)))
		}
	}()

	//j.logger.Info("Starting job")
	account, err := j.RandomAccount()
	if err != nil {
		return err
	}

	awsCred, azureCred, err := j.onboardClient.GetSourceFullCred(&httpclient.Context{
		UserRole: api.KeibiAdminRole,
	}, account.ID.String())
	if err != nil {
		return err
	}

	err = j.PopulateSteampipe(account, awsCred, azureCred)
	if err != nil {
		return err
	}

	cmd := exec.Command("steampipe", "service", "stop")
	err = cmd.Run()
	if err != nil {
		return err
	}
	time.Sleep(5 * time.Second)
	//fmt.Println("+++++ Steampipe service stoped")

	cmd = exec.Command("steampipe", "service", "start", "--database-listen", "network", "--database-port",
		"9193", "--database-password", "abcd")
	err = cmd.Run()
	if err != nil {
		return err
	}
	time.Sleep(5 * time.Second)
	//fmt.Println("+++++ Steampipe service started")

	s1, err := steampipe.NewSteampipeDatabase(steampipe.Option{
		Host: "localhost",
		Port: "9193",
		User: "steampipe",
		Pass: "abcd",
		Db:   "steampipe",
	})
	if err != nil {
		return err
	}
	j.steampipe = s1
	//fmt.Println("+++++ Connected to steampipe")
	query := j.RandomQuery(account.Type)

	j.logger.Info("running query", zap.String("account", account.ConnectionID), zap.String("query", query.ListQuery))
	listQuery := strings.ReplaceAll(query.ListQuery, "%ACCOUNT_ID%", account.ConnectionID)
	listQuery = strings.ReplaceAll(listQuery, "%KEIBI_ACCOUNT_ID%", account.ID.String())
	steampipeRows, err := j.steampipe.Conn().Query(context.Background(), listQuery)
	if err != nil {
		return err
	}
	defer steampipeRows.Close()

	rowCount := 0
	for steampipeRows.Next() {
		rowCount++
		steampipeRow, err := steampipeRows.Values()
		if err != nil {
			return err
		}

		steampipeRecord := map[string]interface{}{}
		for idx, field := range steampipeRows.FieldDescriptions() {
			steampipeRecord[string(field.Name)] = steampipeRow[idx]
		}

		getQuery := strings.ReplaceAll(query.GetQuery, "%ACCOUNT_ID%", account.ConnectionID)
		getQuery = strings.ReplaceAll(getQuery, "%KEIBI_ACCOUNT_ID%", account.ID.String())

		var keyValues []interface{}
		for _, f := range query.KeyFields {
			keyValues = append(keyValues, steampipeRecord[f])
		}

		esRows, err := j.esSteampipe.Conn().Query(context.Background(), getQuery, keyValues...)
		if err != nil {
			return err
		}

		found := false

		for esRows.Next() {
			esRow, err := esRows.Values()
			if err != nil {
				return err
			}

			found = true

			esRecord := map[string]interface{}{}
			for idx, field := range esRows.FieldDescriptions() {
				esRecord[string(field.Name)] = esRow[idx]
			}

			for k, v := range steampipeRecord {
				v2 := esRecord[k]

				j1, err := json.Marshal(v)
				if err != nil {
					return err
				}

				j2, err := json.Marshal(v2)
				if err != nil {
					return err
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
					if k != "etag" && k != "tags" {
						j.logger.Warn("inconsistency in data",
							zap.String("accountID", account.ConnectionID),
							zap.String("steampipe", sj1),
							zap.String("es", sj2),
							zap.String("conflictColumn", k),
							zap.String("keyColumns", fmt.Sprintf("%v", keyValues)),
						)
					}
				}
			}
		}

		if !found {
			j.logger.Warn("record not found",
				zap.String("accountID", account.ConnectionID),
				zap.String("keyColumns", fmt.Sprintf("%v", keyValues)),
			)
		}
	}

	j.logger.Info("Done", zap.Int("rowCount", rowCount))

	return nil
}

func (j *Job) RandomAccount() (*api2.Source, error) {
	srcs, err := j.onboardClient.ListSources(&httpclient.Context{
		UserRole: api.AdminRole,
	}, nil)
	if err != nil {
		return nil, err
	}

	idx := rand.Intn(len(srcs))
	return &srcs[idx], nil
}

func (j *Job) RandomQuery(sourceType source.Type) *Query {
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

func (j *Job) PopulateSteampipe(account *api2.Source, cred *api2.AWSCredential, azureCred *api2.AzureCredential) error {
	dirname, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	filePath := dirname + "/.steampipe/config/steampipe.spc"
	os.MkdirAll(filepath.Dir(filePath), os.ModePerm)

	if cred != nil {
		credFilePath := dirname + "/.aws/credentials"
		os.MkdirAll(filepath.Dir(credFilePath), os.ModePerm)

		content := fmt.Sprintf(`
[default]
aws_access_key_id = %s
aws_secret_access_key = %s
`,
			cred.AccessKey, cred.SecretKey)
		err = os.WriteFile(credFilePath, []byte(content), os.ModePerm)
		if err != nil {
			return err
		}

		//os.Setenv("AWS_ACCESS_KEY_ID", cred.AccessKey)
		//os.Setenv("AWS_SECRET_ACCESS_KEY", cred.SecretKey)
		content = `
connection "aws" {
  plugin  = "aws"
  regions = ["*"]
}
`
		filePath = dirname + "/.steampipe/config/aws.spc"
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
			azureCred.TenantID, account.ConnectionID, azureCred.ClientID, azureCred.ClientSecret)
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
			azureCred.TenantID, azureCred.ClientID, azureCred.ClientSecret)
		filePath = dirname + "/.steampipe/config/azuread.spc"
		return os.WriteFile(filePath, []byte(content), os.ModePerm)
	}

	return nil
}
