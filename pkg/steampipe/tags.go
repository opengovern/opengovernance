package steampipe

import (
	"encoding/json"
	"fmt"

	"github.com/turbot/steampipe-plugin-sdk/grpc/proto"
)

func ExtractTags(resourceType string, source interface{}) (map[string]string, error) {
	var cells map[string]*proto.Column
	pluginProvider := ExtractPlugin(resourceType)
	pluginTableName := ExtractTableName(resourceType)
	if pluginTableName == "" {
		return nil, fmt.Errorf("cannot find table name for resourceType: %s", resourceType)
	}
	if pluginProvider == SteampipePluginAWS {
		desc, err := ConvertToDescription(resourceType, source)
		if err != nil {
			return nil, err
		}

		cells, err = AWSDescriptionToRecord(desc, pluginTableName)
		if err != nil {
			return nil, err
		}
	} else if pluginProvider == SteampipePluginAzure || pluginProvider == SteampipePluginAzureAD {
		if pluginProvider == SteampipePluginAzure {
			desc, err := ConvertToDescription(resourceType, source)
			if err != nil {
				return nil, err
			}

			cells, err = AzureDescriptionToRecord(desc, pluginTableName)
			if err != nil {
				return nil, err
			}
		} else {
			desc, err := ConvertToDescription(resourceType, source)
			if err != nil {
				return nil, err
			}

			cells, err = AzureADDescriptionToRecord(desc, pluginTableName)
			if err != nil {
				return nil, err
			}
		}
	} else {
		return nil, fmt.Errorf("invalid provider for resource type: %s", resourceType)
	}

	tags := map[string]string{}
	for k, v := range cells {
		if k == "tags" {
			if jsonBytes := v.GetJsonValue(); jsonBytes != nil {
				err := json.Unmarshal(jsonBytes, &tags)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return tags, nil
}
