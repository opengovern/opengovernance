package opengovernance_client

import (
	"context"
	"encoding/json"
	"runtime"
	"strings"

	"github.com/opengovern/og-util/pkg/source"
	"github.com/opengovern/opengovernance/pkg/steampipe-plugin-opengovernance/opengovernance-sdk/config"
	"github.com/opengovern/opengovernance/pkg/steampipe-plugin-opengovernance/opengovernance-sdk/pg"
	onboard "github.com/opengovern/opengovernance/services/integration/model"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
)

type ConnectionRow struct {
	KaytuID        string              `json:"kaytu_id"`
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	Connector      source.Type         `json:"connector"`
	HealthState    string              `json:"health_state"`
	LifecycleState string              `json:"lifecycle_state"`
	Tags           map[string][]string `json:"tags"`
}

func getAWSConnectionRowFromConnection(ctx context.Context, connection onboard.Connection) (*ConnectionRow, error) {
	row := ConnectionRow{
		KaytuID:        connection.ID.String(),
		ID:             connection.SourceId,
		Name:           connection.Name,
		Connector:      connection.Type,
		HealthState:    strings.ToLower(string(connection.HealthState)),
		LifecycleState: strings.ToLower(string(connection.LifecycleState)),
		Tags:           make(map[string][]string),
	}

	if connection.Metadata != nil && len(connection.Metadata) != 0 {
		var metadata map[string]any
		err := json.Unmarshal(connection.Metadata, &metadata)
		if err != nil {
			plugin.Logger(ctx).Error("failed to unmarshal aws metadata", "error", err)
			return nil, err
		}
		if tags, ok := metadata["organization_tags"]; ok {
			if tagsMap, ok := tags.(map[string]any); ok {
				for key, value := range tagsMap {
					if strValue, ok := value.(string); ok {
						row.Tags[key] = []string{strValue}
					}
				}
			}
		}
	}

	return &row, nil
}

func getAzureConnectionRowFromConnection(ctx context.Context, connection onboard.Connection) (*ConnectionRow, error) {
	row := ConnectionRow{
		KaytuID:        connection.ID.String(),
		ID:             connection.SourceId,
		Name:           connection.Name,
		Connector:      connection.Type,
		HealthState:    strings.ToLower(string(connection.HealthState)),
		LifecycleState: strings.ToLower(string(connection.LifecycleState)),
		Tags:           make(map[string][]string),
	}

	if connection.Metadata != nil && len(connection.Metadata) != 0 {
		var metadata map[string]any
		err := json.Unmarshal(connection.Metadata, &metadata)
		if err != nil {
			plugin.Logger(ctx).Error("failed to unmarshal azure metadata", "error", err)
			return nil, err
		}

		if tags, ok := metadata["subscription_tags"]; ok {
			if tagsMap, ok := tags.(map[string]any); ok {
				for key, value := range tagsMap {
					if arrValues, ok := value.([]any); ok {
						for _, arrValue := range arrValues {
							if strValue, ok := arrValue.(string); ok {
								row.Tags[key] = append(row.Tags[key], strValue)
							}
						}
					}
				}
			}
		}
	}

	return &row, nil
}

func getConnectionRowFromConnection(ctx context.Context, connection onboard.Connection) (*ConnectionRow, error) {
	switch connection.Type {
	case source.CloudAWS:
		return getAWSConnectionRowFromConnection(ctx, connection)
	case source.CloudAzure:
		return getAzureConnectionRowFromConnection(ctx, connection)
	}
	return nil, nil
}

func ListConnections(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (any, error) {
	plugin.Logger(ctx).Trace("ListConnections")
	runtime.GC()
	cfg := config.GetConfig(d.Connection)
	ke, err := pg.NewClientCached(cfg, d.ConnectionCache, ctx)
	if err != nil {
		return nil, err
	}
	k := Client{PG: ke}

	connections, err := k.PG.ListConnections(ctx)
	if err != nil {
		return nil, err
	}

	for _, connection := range connections {
		row, err := getConnectionRowFromConnection(ctx, connection)
		if err != nil {
			plugin.Logger(ctx).Error("ListConnections", "connection", connection, "error", err)
			continue
		}
		d.StreamListItem(ctx, row)
	}

	return nil, nil
}

func GetConnection(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (any, error) {
	plugin.Logger(ctx).Trace("ListConnections")
	runtime.GC()
	cfg := config.GetConfig(d.Connection)
	ke, err := pg.NewClientCached(cfg, d.ConnectionCache, ctx)
	if err != nil {
		return nil, err
	}
	k := Client{PG: ke}

	kaytuId := d.EqualsQuals["kaytu_id"].GetStringValue()
	id := d.EqualsQuals["id"].GetStringValue()
	connection, err := k.PG.GetConnectionByIDs(ctx, kaytuId, id)
	if err != nil {
		return nil, err
	}
	if connection == nil {
		return nil, nil
	}

	row, err := getConnectionRowFromConnection(ctx, *connection)
	if err != nil {
		plugin.Logger(ctx).Error("GetConnection", "connection", connection, "error", err)
		return nil, err
	}
	return row, nil
}
