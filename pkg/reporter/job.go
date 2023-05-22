package reporter

import (
	"context"
	"fmt"
	"github.com/kaytu-io/kaytu-aws-describer/aws"
	awsSteampipe "github.com/kaytu-io/kaytu-aws-describer/pkg/steampipe"
	"github.com/kaytu-io/kaytu-azure-describer/azure"
	azureSteampipe "github.com/kaytu-io/kaytu-azure-describer/pkg/steampipe"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/config"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	onboardClient "gitlab.com/keibiengine/keibi-engine/pkg/onboard/client"
	"go.uber.org/zap"
	"math/rand"
)

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

	logger, err := zap.NewProduction()
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

func (j *Job) Run() error {
	defer func() {
		if r := recover(); r != nil {
			j.logger.Error("panic", zap.Error(fmt.Errorf("%v", r)))
		}
	}()

	j.logger.Info("Starting job")
	accountID, err := j.RandomAccount()
	if err != nil {
		j.logger.Error("Failed to get account", zap.Error(err))
		return err
	}
	j.logger.Debug("got the account",
		zap.String("accountID", accountID))

	resourceType := j.RandomResourceType()

	j.logger.Debug("got the resource type",
		zap.String("resourceType", resourceType))

	listQuery := j.BuildListQuery(accountID, resourceType)

	j.logger.Debug("query steampipe",
		zap.String("accountID", accountID),
		zap.String("resourceType", resourceType),
		zap.String("query", listQuery))

	steampipeRows, err := j.steampipe.Conn().Query(context.Background(), listQuery)
	if err != nil {
		return err
	}
	defer steampipeRows.Close()

	//TODO-Saleh
	keyFields := []string{"arn"}

	getQuery := j.BuildGetQuery(accountID, resourceType, keyFields)
	for steampipeRows.Next() {
		steampipeRow, err := steampipeRows.Values()
		if err != nil {
			return err
		}

		steampipeRecord := map[string]interface{}{}
		for idx, field := range steampipeRows.FieldDescriptions() {
			steampipeRecord[string(field.Name)] = steampipeRow[idx]
		}

		var keyValues []interface{}
		for _, f := range keyFields {
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

				if v != v2 {
					j.logger.Error("inconsistency in data",
						zap.String("accountID", accountID),
						zap.String("resourceType", resourceType),
						zap.String("steampipeARN", fmt.Sprintf("%v", steampipeRecord["arn"])),
						zap.String("esARN", fmt.Sprintf("%v", esRecord["arn"])),
						zap.String("conflictColumn", k),
					)
				}
			}
		}

		if !found {
			j.logger.Error("record not found",
				zap.String("accountID", accountID),
				zap.String("resourceType", resourceType),
				zap.String("steampipeARN", fmt.Sprintf("%v", steampipeRecord["arn"])),
			)
		}
	}

	return nil
}

func (j *Job) RandomAccount() (string, error) {
	srcs, err := j.onboardClient.ListSources(&httpclient.Context{
		UserRole: api.AdminRole,
	}, nil)
	if err != nil {
		return "", err
	}

	idx := rand.Intn(len(srcs))
	return srcs[idx].ID.String(), nil
}

func (j *Job) RandomResourceType() string {
	var resourceTypes []string
	resourceTypes = append(resourceTypes, aws.ListResourceTypes()...)
	resourceTypes = append(resourceTypes, azure.ListResourceTypes()...)
	idx := rand.Intn(len(resourceTypes))
	return resourceTypes[idx]
}

func (j *Job) BuildListQuery(accountID, resourceType string) string {
	var tableName string

	switch steampipe.ExtractPlugin(resourceType) {
	case steampipe.SteampipePluginAWS:
		tableName = awsSteampipe.ExtractTableName(resourceType)
	case steampipe.SteampipePluginAzure, steampipe.SteampipePluginAzureAD:
		tableName = azureSteampipe.ExtractTableName(resourceType)
	}
	return fmt.Sprintf("SELECT * FROM %s WHERE account_id = '%s'", tableName, accountID)
}

func (j *Job) BuildGetQuery(accountID, resourceType string, keyFields []string) string {
	var tableName string

	switch steampipe.ExtractPlugin(resourceType) {
	case steampipe.SteampipePluginAWS:
		tableName = awsSteampipe.ExtractTableName(resourceType)
	case steampipe.SteampipePluginAzure, steampipe.SteampipePluginAzureAD:
		tableName = azureSteampipe.ExtractTableName(resourceType)
	}

	var q string
	for _, f := range keyFields {
		q += fmt.Sprintf(" AND %s = ?", f)
	}
	return fmt.Sprintf("SELECT * FROM %s WHERE account_id = '%s' %s", tableName, accountID, q)
}
