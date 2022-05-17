package source

import (
	"errors"
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
		return "", errors.New("invalid provider")
	}
}
