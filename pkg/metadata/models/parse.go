package models

import (
	"encoding/json"
	"strconv"

	metadataErrors "github.com/kaytu-io/kaytu-engine/pkg/metadata/errors"
)

func (t ConfigMetadataType) SerializeValue(value any) (string, error) {
	switch t {
	case ConfigMetadataTypeString:
		valueStr, ok := value.(string)
		if !ok {
			return "", metadataErrors.ErrMetadataValueTypeMismatch
		}
		return valueStr, nil
	case ConfigMetadataTypeInt:
		valueInt, ok := value.(int)
		if !ok {
			return "", metadataErrors.ErrMetadataValueTypeMismatch
		}
		return strconv.Itoa(valueInt), nil
	case ConfigMetadataTypeBool:
		valueBool, ok := value.(bool)
		if !ok {
			return "", metadataErrors.ErrMetadataValueTypeMismatch
		}
		return strconv.FormatBool(valueBool), nil
	case ConfigMetadataTypeJSON:
		valueJson, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		return string(valueJson), nil
	}
	return "", metadataErrors.ErrMetadataValueTypeMismatch
}

func (t ConfigMetadataType) DeserializeValue(value string) (any, error) {
	switch t {
	case ConfigMetadataTypeString:
		return value, nil
	case ConfigMetadataTypeInt:
		valueInt, err := strconv.Atoi(value)
		if err != nil {
			return nil, err
		}
		return valueInt, nil
	case ConfigMetadataTypeBool:
		valueBool, err := strconv.ParseBool(value)
		if err != nil {
			return nil, err
		}
		return valueBool, nil
	case ConfigMetadataTypeJSON:
		var valueJson any
		err := json.Unmarshal([]byte(value), &valueJson)
		if err != nil {
			return nil, err
		}
		return valueJson, nil
	}
	return nil, metadataErrors.ErrMetadataValueTypeMismatch
}

func (c *ConfigMetadata) ParseToType() (IConfigMetadata, error) {
	value, err := c.Type.DeserializeValue(c.Value)
	if err != nil {
		return nil, err
	}
	switch c.Type {
	case ConfigMetadataTypeString:
		return &StringConfigMetadata{ConfigMetadata: *c}, nil
	case ConfigMetadataTypeInt:
		return &IntConfigMetadata{ConfigMetadata: *c, Value: value.(int)}, nil
	case ConfigMetadataTypeBool:
		return &BoolConfigMetadata{ConfigMetadata: *c, Value: value.(bool)}, nil
	case ConfigMetadataTypeJSON:
		return &JSONConfigMetadata{ConfigMetadata: *c, Value: value}, nil
	}

	return nil, metadataErrors.ErrConfigMetadataTypeNotSupported
}
