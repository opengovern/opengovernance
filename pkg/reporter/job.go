package reporter

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-aws-describer/aws"
	awsSteampipe "github.com/kaytu-io/kaytu-aws-describer/pkg/steampipe"
	"github.com/kaytu-io/kaytu-azure-describer/azure"
	azureSteampipe "github.com/kaytu-io/kaytu-azure-describer/pkg/steampipe"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/config"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	api2 "gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"
	onboardClient "gitlab.com/keibiengine/keibi-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"go.uber.org/zap"
	"math/rand"
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
	Steampipe   config.Postgres
	SteampipeES config.Postgres
	Onboard     config.KeibiService
}

type Job struct {
	steampipe     *steampipe.Database
	esSteampipe   *steampipe.Database
	onboardClient onboardClient.OnboardServiceClient
	logger        *zap.Logger
}

func New(config JobConfig) (*Job, error) {
	if err := json.Unmarshal([]byte(awsQueriesStr), &awsQueries); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(azureQueriesStr), &azureQueries); err != nil {
		return nil, err
	}

	s1, err := steampipe.NewSteampipeDatabase(steampipe.Option{
		Host: config.Steampipe.Host,
		Port: config.Steampipe.Port,
		User: config.Steampipe.Username,
		Pass: config.Steampipe.Password,
		Db:   config.Steampipe.DB,
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

	return &Job{
		steampipe:     s1,
		esSteampipe:   s2,
		onboardClient: onboard,
		logger:        logger,
	}, nil
}

func (j *Job) Run() {
	fmt.Println("starting scheduling")
	for {
		fmt.Println("starting job")
		if err := j.RunJob(); err != nil {
			j.logger.Error("failed to run job", zap.Error(err))
		}
		time.Sleep(5 * time.Minute)
	}
}

func (j *Job) RunJob() error {
	defer func() {
		if r := recover(); r != nil {
			j.logger.Error("panic", zap.Error(fmt.Errorf("%v", r)))
		}
	}()

	j.logger.Info("Starting job")
	account, err := j.RandomAccount()
	if err != nil {
		return err
	}

	query := j.RandomQuery(account.Type)
	j.logger.Info("query steampipe",
		zap.String("accountID", account.ConnectionID),
		zap.String("query", query.ListQuery))

	listQuery := strings.ReplaceAll(query.ListQuery, "%ACCOUNT_ID%", account.ID.String())
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

		getQuery := strings.ReplaceAll(query.GetQuery, "%ACCOUNT_ID%", account.ID.String())

		var keyValues []interface{}
		for _, f := range query.KeyFields {
			keyValues = append(keyValues, steampipeRecord[f])
		}

		j.logger.Info("query steampipe",
			zap.String("getQuery", getQuery), zap.String("keyValues", fmt.Sprintf("%v", keyValues)))
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

				if v != v2 {
					j.logger.Error("inconsistency in data",
						zap.String("accountID", account.ConnectionID),
						zap.String("steampipeARN", fmt.Sprintf("%v", steampipeRecord["arn"])),
						zap.String("esARN", fmt.Sprintf("%v", esRecord["arn"])),
						zap.String("conflictColumn", k),
					)
				}
			}
		}

		if !found {
			j.logger.Error("record not found",
				zap.String("accountID", account.ConnectionID),
				zap.String("steampipeARN", fmt.Sprintf("%v", steampipeRecord["arn"])),
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

func (j *Job) RandomTableName(sourceType source.Type) string {
	var resourceTypes []string
	switch sourceType {
	case source.CloudAWS:
		resourceTypes = append(resourceTypes, aws.ListResourceTypes()...)
	case source.CloudAzure:
		resourceTypes = append(resourceTypes, azure.ListResourceTypes()...)
	}
	idx := rand.Intn(len(resourceTypes))
	resourceType := resourceTypes[idx]
	var tableName string
	switch steampipe.ExtractPlugin(resourceType) {
	case steampipe.SteampipePluginAWS:
		tableName = awsSteampipe.ExtractTableName(resourceType)
	case steampipe.SteampipePluginAzure, steampipe.SteampipePluginAzureAD:
		tableName = azureSteampipe.ExtractTableName(resourceType)
	}

	if tableName == "" {
		return j.RandomTableName(sourceType)
	}
	return tableName
}

func (j *Job) BuildListQuery(account *api2.Source, tableName string) string {
	return fmt.Sprintf("SELECT * FROM %s WHERE keibi_account_id = '%s'", tableName, account.ID.String())
}

func (j *Job) BuildGetQuery(account *api2.Source, tableName string, keyFields []string) string {
	var q string
	c := 1
	for _, f := range keyFields {
		q += fmt.Sprintf(" AND %s = $%d", f, c)
		c++
	}
	return fmt.Sprintf("SELECT * FROM %s WHERE keibi_account_id = '%s' %s", tableName, account.ID.String(), q)
}
