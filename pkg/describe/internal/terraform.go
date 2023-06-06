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
	if arns[0] == "/" { // Azure
		for _, arn := range arns {
			splitArn := strings.Split(arn, "/")
			if _, value := haveAcc[splitArn[2]]; !value && splitArn[2] != "" {
				haveAcc[splitArn[2]] = true
				accounts = append(accounts, splitArn[2])
			}
		}
	} else { // AWS
		for _, arn := range arns {
			splitArn := strings.Split(arn, ":")
			if _, value := haveAcc[splitArn[4]]; !value && splitArn[4] != "" {
				haveAcc[splitArn[4]] = true
				accounts = append(accounts, splitArn[4])
			}
		}
	}
	return accounts, nil
}
