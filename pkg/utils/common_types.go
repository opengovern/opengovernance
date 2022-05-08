package utils

import (
	"errors"
	"strings"
)

type SourceType string

const (
	SourceCloudAWS   SourceType = "AWS"
	SourceCloudAzure SourceType = "Azure"
)

func ParseSourceType(str string) (SourceType, error) {
	str = strings.ToLower(str)
	switch str {
	case "aws":
		return SourceCloudAWS, nil
	case "azure":
		return SourceCloudAzure, nil
	default:
		return "", errors.New("invalid provider")
	}
}
