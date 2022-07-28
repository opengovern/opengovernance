package source

import (
	"fmt"
	"strings"
)

type Type string

const (
	CloudAWS   Type = "AWS"
	CloudAzure Type = "Azure"
)

func ParseType(str string) (Type, error) {
	str = strings.ToLower(str)
	switch str {
	case "aws":
		return CloudAWS, nil
	case "azure":
		return CloudAzure, nil
	default:
		return "", fmt.Errorf("invalid provider: %s", str)
	}
}
