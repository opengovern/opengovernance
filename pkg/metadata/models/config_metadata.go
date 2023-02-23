package models

type IConfigMetadata interface {
	GetKey() MetadataKey
	GetType() ConfigMetadataType
	GetValue() any
	GetCore() ConfigMetadata
}

type ConfigMetadata struct {
	Key   MetadataKey        `json:"key" gorm:"primary_key"`
	Type  ConfigMetadataType `json:"type" gorm:"default:'string'"`
	Value string             `json:"value" gorm:"type:text;not null"`
}

type StringConfigMetadata struct {
	ConfigMetadata
}

func (c *StringConfigMetadata) GetKey() MetadataKey {
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

type IntConfigMetadata struct {
	ConfigMetadata
	Value int
}

func (c *IntConfigMetadata) GetKey() MetadataKey {
	return c.Key
}

func (c *IntConfigMetadata) GetType() ConfigMetadataType {
	return ConfigMetadataTypeInt
}

func (c *IntConfigMetadata) GetValue() any {
	return c.Value
}

func (c *IntConfigMetadata) GetCore() ConfigMetadata {
	return c.ConfigMetadata
}

type BoolConfigMetadata struct {
	ConfigMetadata
	Value bool
}

func (c *BoolConfigMetadata) GetKey() MetadataKey {
	return c.Key
}

func (c *BoolConfigMetadata) GetType() ConfigMetadataType {
	return ConfigMetadataTypeBool
}

func (c *BoolConfigMetadata) GetValue() any {
	return c.Value
}

func (c *BoolConfigMetadata) GetCore() ConfigMetadata {
	return c.ConfigMetadata
}

type JSONConfigMetadata struct {
	ConfigMetadata
	Value any
}

func (c *JSONConfigMetadata) GetKey() MetadataKey {
	return c.Key
}

func (c *JSONConfigMetadata) GetType() ConfigMetadataType {
	return ConfigMetadataTypeJSON
}

func (c *JSONConfigMetadata) GetValue() any {
	return c.Value
}

func (c *JSONConfigMetadata) GetCore() ConfigMetadata {
	return c.ConfigMetadata
}
