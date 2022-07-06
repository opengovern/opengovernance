package steampipe

import (
	"encoding/json"
	"errors"

	"github.com/turbot/steampipe-plugin-sdk/grpc/proto"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
)

func ExtractTags(resourceType string, description interface{}) (map[string]string, error) {
	var cells map[string]*proto.Column
	pluginProvider := ExtractPlugin(resourceType)
	pluginTableName := ExtractTableName(resourceType)
	if pluginProvider == SteampipePluginAWS {
		desc, err := api.ConvertToDescription(resourceType, description)
		if err != nil {
			return nil, err
		}

		cells, err = AWSDescriptionToRecord(desc, pluginTableName)
		if err != nil {
			return nil, err
		}
	} else if pluginProvider == SteampipePluginAzure || pluginProvider == SteampipePluginAzureAD {
		desc, err := api.ConvertToDescription(resourceType, description)
		if err != nil {
			return nil, err
		}

		if pluginProvider == SteampipePluginAzure {
			cells, err = AzureDescriptionToRecord(desc, pluginTableName)
			if err != nil {
				return nil, err
			}
		} else {
			cells, err = AzureADDescriptionToRecord(desc, pluginTableName)
			if err != nil {
				return nil, err
			}
		}
	} else {
		return nil, errors.New("invalid provider")
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
