package internal

import (
	"io"
	"strings"

	"github.com/kaytu-io/terraform-package/external/states/statefile"
)

func GetArns(content string) ([]string, error) {
	reader := io.Reader(strings.NewReader(content))
	arns := statefile.GetResourcesArn(reader)
	return arns, nil
}

func getTypes(content string) ([]string, error) {
	reader := io.Reader(strings.NewReader(content))
	types := statefile.GetResourcesTypes(reader)
	return types, nil
}

func ParseAccountsFromArns(arns []string) ([]string, error) {
	haveAcc := make(map[string]bool)
	var accounts []string
	for _, arn := range arns {
		if arn[0] == '/' { // Azure
			splitArn := strings.Split(arn, "/")
			acc := splitArn[2]
			if _, value := haveAcc[acc]; !value && acc != "" {
				haveAcc[acc] = true
				accounts = append(accounts, acc)
			}
		} else { // AWS
			splitArn := strings.Split(arn, ":")
			acc := splitArn[4]
			if _, value := haveAcc[acc]; !value && acc != "" {
				haveAcc[acc] = true
				accounts = append(accounts, acc)
			}
		}
	}
	return accounts, nil
}

func GetResourceIDFromArn(arns []string) ([]string, error) {
	haveResource := make(map[string]bool)
	var resources []string
	for _, arn := range arns {
		if arn[0] == '/' { //Azure
			resources = append(resources, arn)
		} else { // AWS
			splitArn := strings.Split(arn, ":")
			res := strings.Split(splitArn[5], "/")[1]
			if _, value := haveResource[res]; !value && res != "" {
				haveResource[res] = true
				resources = append(resources, res)
			}
		}
	}
	return resources, nil
}
