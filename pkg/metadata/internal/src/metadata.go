package src

import (
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
	"gitlab.com/keibiengine/keibi-engine/pkg/metadata/internal/cache"
	"gitlab.com/keibiengine/keibi-engine/pkg/metadata/internal/database"
	"gitlab.com/keibiengine/keibi-engine/pkg/metadata/models"
)

const (
	ConfigMetadataKeyPrefix = "config_metadata:"
)

func GetConfigMetadata(db database.Database, rdb *cache.MetadataRedisCache, key string) (models.IConfigMetadata, error) {
	value, err := rdb.Get(ConfigMetadataKeyPrefix + key)
	if err == nil {
		var cm models.ConfigMetadata
		err := json.Unmarshal([]byte(value), &cm)
		if err != nil {
			return nil, err
		}
		typedCm, err := cm.ParseToType()
		if err != nil {
			return nil, err
		}
		return typedCm, nil
	} else if err != redis.Nil {
		fmt.Printf("error getting config metadata from redis: %v\n", err)
	}

	typedCm, err := db.GetConfigMetadata(key)
	if err != nil {
		return nil, err
	}
	jsonCm, err := json.Marshal(typedCm.GetCore())
	if err != nil {
		fmt.Printf("error marshalling config metadata: %v\n", err)
		return typedCm, nil
	}

	err = rdb.Set(ConfigMetadataKeyPrefix+key, string(jsonCm))
	if err != nil {
		fmt.Printf("error setting config metadata in redis: %v\n", err)
		return typedCm, nil
	}

	return typedCm, nil
}

func SetConfigMetadata(db database.Database, rdb *cache.MetadataRedisCache, key string, value any) error {
	cmType := models.GetConfigMetadataTypeFromKey(key)
	valueStr := ""
	switch cmType {
	case models.ConfigMetadataTypeString:
		valueStr = value.(string)
	}
	err := db.SetConfigMetadata(models.ConfigMetadata{
		Key:   key,
		Type:  cmType,
		Value: valueStr,
	})
	if err != nil {
		return err
	}
	err = rdb.Delete(ConfigMetadataKeyPrefix + key)
	if err != nil && err != redis.Nil {
		fmt.Printf("error deleting config metadata from redis: %v\n", err)
	}
	return nil
}
