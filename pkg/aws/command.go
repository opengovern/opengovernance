package aws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	var regions []string
	var resourceType string
	var awsAccount string
	var awsSecretKey string
	var awsAccessKey string
	var awsSessionToken string
	var disabledRegions bool
	var assumeRoleArn string

	cmd := &cobra.Command{
		Use: "aws",
		Example: `
# Query the list of EC2 Instances in us-east-1:

	cloud-inventory aws --type AWS::EC2::Instance --regions us-east-1 --account-id 12314214324

# Query the list of EC2 VPCs in us-east-1 and us-west-2:

	cloud-inventory aws --type AWS::EC2::VPC --regions us-east-1,us-west-2 --account-id 12314214324

# Query the list of EC2 Subnets in all regions:

	cloud-inventory aws --type AWS::EC2::Subnet --account-id 12314214324

# Query the list of EC2 Instances using the provided AccessKey and SecretKey:

	cloud-inventory aws --type AWS::EC2::Instance --account-id 12314214324 --secret-key 1fadsrqendfq3ud --access-key feqfefedff23

# Query the list of SecurityGroups by assuming the role SecurityAuditorRole

	cloud-inventory aws --type AWS::EC2::Instance --account-id 12314214324 --secret-key 1fadsrqendfq3ud --access-key feqfefedff23 --assume-role-arn arn:aws:iam::123456789012:role/SecurityAuditorRole

`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case awsAccount == "":
				return errors.New("required flag 'account-id' has not been set")
			case resourceType == "":
				return errors.New("required flag 'type' has not been set")
			case (awsAccessKey == "") != (awsSecretKey == ""):
				return errors.New("flags 'access-key' and 'secret-key' must be either both set or left empty")
			case len(regions) > 0 && disabledRegions:
				return errors.New("flag 'include-disabled-regions' can't be set while regions are specified explicitly")
			default:
				return nil
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// After this, if we return error, don't print out usage. The error is a runtime error.
			cmd.SilenceUsage = true

			ctx := context.Background()
			cfg, err := getConfig(ctx, awsAccessKey, awsSecretKey, awsSessionToken, assumeRoleArn)
			if err != nil {
				return err
			}

			if len(regions) == 0 {
				rs, err := GetAllRegions(ctx, cfg, disabledRegions)
				if err != nil {
					return err
				}

				for _, r := range rs {
					regions = append(regions, *r.RegionName)
				}
			}

			response, err := GetResources(ctx, cfg, regions, resourceType)
			if err != nil {
				return err
			}

			output := map[string]interface{}{
				"resources": response.Resources,
				"errors":    response.Errors,
				"metadata": map[string]interface{}{
					"type":      resourceType,
					"accountId": awsAccount,
					"regions":   regions,
				},
			}

			bytes, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				return err
			}

			fmt.Println(string(bytes))
			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&resourceType, "type", "t", "", "AWS Resource Type, e.g. 'AWS::EC2::Instance'")
	cmd.PersistentFlags().StringSliceVarP(&regions, "regions", "r", []string{},
		`Comma seperated list of regions to query, e.g. 'us-east-1,us-east-2'. 
If no region is specified, the resource will be queries from all AWS regions`)
	cmd.PersistentFlags().BoolVar(&disabledRegions, "include-disabled-regions", false, "By default, regions such as ")
	cmd.PersistentFlags().StringVar(&awsAccount, "account-id", "", "AWS Account id")
	cmd.PersistentFlags().StringVar(&awsSecretKey, "secret-key", "", "AWS SecretKey from the credentials. If not specified, the defailt shared aws config will be used")
	cmd.PersistentFlags().StringVar(&awsAccessKey, "access-key", "", "AWS AccessKey from the credentials. If not specified, the defailt shared aws config will be used")
	cmd.PersistentFlags().StringVar(&assumeRoleArn, "assume-role-arn", "",
		`AWS AssumeRoleARN, e.g. 'arn:aws:iam::123456789012:role/roleName'.
If provided, will assume the role with the provided credentials.
If no credentials are provided the default AWS credentials.
The role must have access to describe the specific resource type as
well as 'EC2:DescribeRegions'`)
	cmd.PersistentFlags().StringVar(&awsSessionToken, "session-token", "", "AWS SessionToken from the credentials")

	cmd.AddCommand(listResourcesCommand())

	return cmd
}

func listResourcesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "list-resources",
		Run: func(cmd *cobra.Command, args []string) {
			for k := range ResourceTypeToDescriber {
				fmt.Println(k)
			}
		},
	}

	return cmd
}
