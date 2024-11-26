package models

import (
	"encoding/json"
	"strconv"

	metadataErrors "github.com/opengovern/opencomply/services/metadata/errors"
)

func (t ConfigMetadataType) SerializeValue(value any) (string, error) {
	switch t {
	case ConfigMetadataTypeString:
		valueStr, ok := value.(string)
		if !ok {
			return "", metadataErrors.ErrorMetadataValueTypeMismatch
		}
		return valueStr, nil
	case ConfigMetadataTypeInt:
		switch value.(type) {
		case int:
			return strconv.Itoa(value.(int)), nil
		case string:
			valueM, err := strconv.ParseInt(value.(string), 10, 64)
			if err != nil {
				return "", err
			}
			return strconv.Itoa(int(valueM)), nil
		default:
			return "", metadataErrors.ErrorMetadataValueTypeMismatch
		}
	case ConfigMetadataTypeBool:
		switch value.(type) {
		case bool:
			return strconv.FormatBool(value.(bool)), nil
		case string:
			valueM, err := strconv.ParseBool(value.(string))
			if err != nil {
				return "", err
			}
			return strconv.FormatBool(valueM), nil
		default:
			return "", metadataErrors.ErrorMetadataValueTypeMismatch
		}
	case ConfigMetadataTypeJSON:
		valueJson, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		return string(valueJson), nil
	}
	return "", metadataErrors.ErrorMetadataValueTypeMismatch
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
	return nil, metadataErrors.ErrorMetadataValueTypeMismatch
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

	return nil, metadataErrors.ErrorConfigMetadataTypeNotSupported
}
