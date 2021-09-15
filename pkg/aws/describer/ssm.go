package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

func SSMAssociation(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ssm.NewFromConfig(cfg)
	paginator := ssm.NewListAssociationsPaginator(client, &ssm.ListAssociationsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Associations {
			values = append(values, v)
		}
	}

	return values, nil
}

func SSMDocument(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ssm.NewFromConfig(cfg)
	paginator := ssm.NewListDocumentsPaginator(client, &ssm.ListDocumentsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.DocumentIdentifiers {
			values = append(values, v)
		}
	}

	return values, nil
}

func SSMMaintenanceWindow(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ssm.NewFromConfig(cfg)
	paginator := ssm.NewDescribeMaintenanceWindowsPaginator(client, &ssm.DescribeMaintenanceWindowsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.WindowIdentities {
			values = append(values, v)
		}
	}

	return values, nil
}

func SSMMaintenanceWindowTarget(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ssm.NewFromConfig(cfg)
	paginator := ssm.NewDescribeMaintenanceWindowTargetsPaginator(client, &ssm.DescribeMaintenanceWindowTargetsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Targets {
			values = append(values, v)
		}
	}

	return values, nil
}

func SSMMaintenanceWindowTask(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ssm.NewFromConfig(cfg)
	paginator := ssm.NewDescribeMaintenanceWindowTasksPaginator(client, &ssm.DescribeMaintenanceWindowTasksInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Tasks {
			values = append(values, v)
		}
	}

	return values, nil
}

func SSMParameter(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ssm.NewFromConfig(cfg)
	paginator := ssm.NewDescribeParametersPaginator(client, &ssm.DescribeParametersInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Parameters {
			values = append(values, v)
		}
	}

	return values, nil
}

func SSMPatchBaseline(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ssm.NewFromConfig(cfg)
	paginator := ssm.NewDescribePatchBaselinesPaginator(client, &ssm.DescribePatchBaselinesInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.BaselineIdentities {
			values = append(values, v)
		}
	}

	return values, nil
}

func SSMResourceDataSync(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ssm.NewFromConfig(cfg)
	paginator := ssm.NewListResourceDataSyncPaginator(client, &ssm.ListResourceDataSyncInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ResourceDataSyncItems {
			values = append(values, v)
		}
	}

	return values, nil
}
