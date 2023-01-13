package models

type ConfigMetadataType string

const (
	ConfigMetadataTypeString ConfigMetadataType = "string"
)

var (
	ConfigMetadataKeyToTypeMapping = map[string]ConfigMetadataType{}
)

func GetConfigMetadataTypeFromKey(key string) ConfigMetadataType {
	if cmType, ok := ConfigMetadataKeyToTypeMapping[key]; ok {
		return cmType
	}
	return ConfigMetadataTypeString
}

type IConfigMetadata interface {
	GetKey() string
	GetType() ConfigMetadataType
	GetValue() any
	GetCore() ConfigMetadata
}

type ConfigMetadata struct {
	Key   string             `json:"key" gorm:"primary_key"`
	Type  ConfigMetadataType `json:"type" gorm:"default:'string'"`
	Value string             `json:"value" gorm:"type:text;not null"`
}

type StringConfigMetadata struct {
	ConfigMetadata
}

func (c *StringConfigMetadata) GetKey() string {
	return c.Key
}

func (c *StringConfigMetadata) GetType() ConfigMetadataType {
	return ConfigMetadataTypeString
}

func (c *StringConfigMetadata) GetValue() any {
	return c.Value
}

func (c *StringConfigMetadata) GetCore() ConfigMetadata {
	return c.ConfigMetadata
}
