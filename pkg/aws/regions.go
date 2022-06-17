package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws/endpoints"
)

func CheckEnoughPermission(awsPermissionCheckUrl, accessKey, secretKey string) error {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	body := fmt.Sprintf(`{"access_key_id": "%s", "secret_access_key": "%s"}`, accessKey, secretKey)
	resp, err := client.Post(awsPermissionCheckUrl, "application/json", strings.NewReader(body))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("non-200 status code: %d", resp.StatusCode)
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var js map[string]string
	err = json.Unmarshal(content, &js)
	if err != nil {
		return err
	}

	//accountID := js["account_id"]
	//accountName := js["account_alias"]
	return nil
}

func CheckDescribeRegionsPermission(accessKey, secretKey string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := GetConfig(ctx, accessKey, secretKey, "", "")
	if err != nil {
		return err
	}

	cfgClone := cfg.Copy()
	cfgClone.Region = "us-east-1"

	_, err = getAllRegions(ctx, cfgClone, false)
	if err != nil {
		return err
	}
	return nil
}

func getAllRegions(ctx context.Context, cfg aws.Config, includeDisabledRegions bool) ([]types.Region, error) {
	client := ec2.NewFromConfig(cfg)
	output, err := client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{
		AllRegions: &includeDisabledRegions,
	})
	if err != nil {
		return nil, err
	}

	return output.Regions, nil
}

func partitionOf(region string) (string, bool) {
	resolver := endpoints.DefaultResolver()
	partitions := resolver.(endpoints.EnumPartitions).Partitions()

	for _, p := range partitions {
		for r := range p.Regions() {
			if r == region {
				return p.ID(), true
			}
		}
	}

	return "", false
}
