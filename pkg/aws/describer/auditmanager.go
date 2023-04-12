package describer

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/auditmanager"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func AuditManagerAssessment(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := auditmanager.NewFromConfig(cfg)
	paginator := auditmanager.NewListAssessmentsPaginator(client, &auditmanager.ListAssessmentsInput{})

	//describeCtx := GetDescribeContext(ctx)

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, assessmentMetadataItem := range page.AssessmentMetadata {

			assessment, err := client.GetAssessment(ctx, &auditmanager.GetAssessmentInput{
				AssessmentId: assessmentMetadataItem.Id,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN:  *assessment.Assessment.Arn,
				Name: *assessment.Assessment.Metadata.Name,
				ID:   *assessment.Assessment.Metadata.Id,
				Description: model.AuditManagerAssessmentDescription{
					Assessment: *assessment.Assessment,
				},
			})
		}
	}

	return values, nil
}

func AuditManagerControl(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := auditmanager.NewFromConfig(cfg)
	paginator := auditmanager.NewListControlsPaginator(client, &auditmanager.ListControlsInput{})

	//describeCtx := GetDescribeContext(ctx)

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, controlMetadata := range page.ControlMetadataList {
			control, err := client.GetControl(ctx, &auditmanager.GetControlInput{
				ControlId: controlMetadata.Id,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN:  *control.Control.Arn,
				Name: *control.Control.Name,
				ID:   *control.Control.Id,
				Description: model.AuditManagerControlDescription{
					Control: *control.Control,
				},
			})
		}
	}

	return values, nil
}

func GetAuditManagerControl(ctx context.Context, cfg aws.Config, controlID string) ([]Resource, error) {
	client := auditmanager.NewFromConfig(cfg)
	control, err := client.GetControl(ctx, &auditmanager.GetControlInput{
		ControlId: &controlID,
	})
	if err != nil {
		return nil, err
	}

	var values []Resource
	values = append(values, Resource{
		ARN:  *control.Control.Arn,
		Name: *control.Control.Name,
		ID:   *control.Control.Id,
		Description: model.AuditManagerControlDescription{
			Control: *control.Control,
		},
	})

	return values, nil
}

func AuditManagerEvidence(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := auditmanager.NewFromConfig(cfg)
	paginator := auditmanager.NewListAssessmentsPaginator(client, &auditmanager.ListAssessmentsInput{})

	describeCtx := GetDescribeContext(ctx)

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, assessmentMetadataItem := range page.AssessmentMetadata {
			evidenceFolderPaginator := auditmanager.NewGetEvidenceFoldersByAssessmentPaginator(client, &auditmanager.GetEvidenceFoldersByAssessmentInput{
				AssessmentId: assessmentMetadataItem.Id,
			})

			for evidenceFolderPaginator.HasMorePages() {
				evidenceFolderPage, err := evidenceFolderPaginator.NextPage(ctx)
				if err != nil {
					return nil, err
				}

				for _, evidenceFolder := range evidenceFolderPage.EvidenceFolders {
					evidencePaginator := auditmanager.NewGetEvidenceByEvidenceFolderPaginator(client, &auditmanager.GetEvidenceByEvidenceFolderInput{
						EvidenceFolderId: evidenceFolder.Id,
					})

					for evidencePaginator.HasMorePages() {
						evidencePage, err := evidencePaginator.NextPage(ctx)
						if err != nil {
							return nil, err
						}

						for _, evidence := range evidencePage.Evidence {
							arn := fmt.Sprintf("arn:%s:auditmanager:%s:%s:evidence/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, *evidence.Id)
							values = append(values, Resource{
								ARN: arn,
								ID:  *evidence.Id,
								Description: model.AuditManagerEvidenceDescription{
									Evidence:     evidence,
									ControlSetID: *evidenceFolder.ControlSetId,
									AssessmentID: *assessmentMetadataItem.Id,
								},
							})
						}
					}
				}
			}
		}
	}

	return values, nil
}

func AuditManagerEvidenceFolder(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := auditmanager.NewFromConfig(cfg)
	paginator := auditmanager.NewListAssessmentsPaginator(client, &auditmanager.ListAssessmentsInput{})

	describeCtx := GetDescribeContext(ctx)

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, assessmentMetadataItem := range page.AssessmentMetadata {
			evidenceFolderPaginator := auditmanager.NewGetEvidenceFoldersByAssessmentPaginator(client, &auditmanager.GetEvidenceFoldersByAssessmentInput{
				AssessmentId: assessmentMetadataItem.Id,
			})

			for evidenceFolderPaginator.HasMorePages() {
				evidenceFolderPage, err := evidenceFolderPaginator.NextPage(ctx)
				if err != nil {
					return nil, err
				}

				for _, evidenceFolder := range evidenceFolderPage.EvidenceFolders {
					arn := fmt.Sprintf("arn:%s:auditmanager:%s:%s:evidence-folder/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, *evidenceFolder.Id)
					values = append(values, Resource{
						ARN:  arn,
						Name: *evidenceFolder.Name,
						ID:   *evidenceFolder.Id,
						Description: model.AuditManagerEvidenceFolderDescription{
							EvidenceFolder: evidenceFolder,
							AssessmentID:   *assessmentMetadataItem.Id,
						},
					})
				}
			}
		}
	}

	return values, nil
}

func AuditManagerFramework(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := auditmanager.NewFromConfig(cfg)
	paginator := auditmanager.NewListAssessmentFrameworksPaginator(client, &auditmanager.ListAssessmentFrameworksInput{})

	//describeCtx := GetDescribeContext(ctx)

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, frameworkMetadata := range page.FrameworkMetadataList {
			framework, err := client.GetAssessmentFramework(ctx, &auditmanager.GetAssessmentFrameworkInput{
				FrameworkId: frameworkMetadata.Id,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN:  *framework.Framework.Arn,
				Name: *framework.Framework.Name,
				ID:   *framework.Framework.Id,
				Description: model.AuditManagerFrameworkDescription{
					Framework: *framework.Framework,
				},
			})
		}
	}

	return values, nil
}
