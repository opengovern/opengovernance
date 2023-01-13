package models

import "errors"

var (
	ErrConfigMetadataTypeNotSupported = errors.New("config metadata type not supported")
)

func (c *ConfigMetadata) ParseToType() (IConfigMetadata, error) {
	switch c.Type {
	case ConfigMetadataTypeString:
		return &StringConfigMetadata{ConfigMetadata: *c}, nil
	}

	return nil, ErrConfigMetadataTypeNotSupported
}
