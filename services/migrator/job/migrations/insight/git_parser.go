package insight

import (
	"fmt"
	"github.com/goccy/go-yaml"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/db"
	"github.com/kaytu-io/kaytu-util/pkg/model"
	"gorm.io/gorm"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type GitParser struct {
	queries       []db.Query
	insights      []db.Insight
	insightGroups []db.InsightGroup
}

func (g *GitParser) queryIDs() []string {
	var queryIDs []string
	for _, query := range g.queries {
		queryIDs = append(queryIDs, query.ID)
	}
	return queryIDs
}

func (g *GitParser) ExtractInsights(queryPath string) error {
	return filepath.WalkDir(queryPath, func(path string, d fs.DirEntry, err error) error {
		if strings.HasSuffix(path, ".yaml") {
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failure in reading file: %v", err)
			}

			var insight Insight
			err = yaml.Unmarshal(content, &insight)
			if err != nil {
				return err
			}

			var tags []db.InsightTag
			for key, value := range insight.Tags {
				tags = append(tags, db.InsightTag{
					Tag: model.Tag{
						Key:   key,
						Value: value,
					},
					InsightID: insight.ID,
				})
			}

			g.insights = append(g.insights, db.Insight{
				Model: gorm.Model{
					ID: insight.ID,
				},
				QueryID:     insight.Query.ID,
				Connector:   insight.Connector,
				ShortTitle:  insight.ShortTitle,
				LongTitle:   insight.LongTitle,
				Description: insight.Description,
				LogoURL:     insight.LogoURL,
				Tags:        tags,
				Links:       insight.Links,
				Enabled:     insight.Enabled,
				Internal:    insight.Internal,
			})
			insight.Query.ID = fmt.Sprintf("insight_%d", insight.ID)
			q := db.Query{
				ID:             insight.Query.ID,
				QueryToExecute: insight.Query.QueryToExecute,
				Connector:      insight.Connector.String(),
				PrimaryTable:   insight.Query.PrimaryTable,
				ListOfTables:   insight.Query.ListOfTables,
				Engine:         insight.Query.Engine,
			}
			for _, p := range insight.Query.Parameters {
				q.Parameters = append(q.Parameters, db.QueryParameter{
					QueryID:  q.ID,
					Key:      p.Key,
					Required: p.Required,
				})
			}

			g.queries = append(g.queries, q)
		}

		return nil
	})
}

func (g *GitParser) ExtractInsightGroups(queryPath string) error {
	return filepath.WalkDir(queryPath, func(path string, d fs.DirEntry, err error) error {
		if strings.HasSuffix(path, ".yaml") {
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failure in reading file: %v", err)
			}

			var insightGroup InsightGroup
			err = yaml.Unmarshal(content, &insightGroup)
			if err != nil {
				return err
			}

			insights := make([]db.Insight, 0, len(insightGroup.InsightIDs))
			for _, insightID := range insightGroup.InsightIDs {
				insights = append(insights, db.Insight{
					Model: gorm.Model{
						ID: insightID,
					},
				})
			}

			g.insightGroups = append(g.insightGroups, db.InsightGroup{
				Model: gorm.Model{
					ID: insightGroup.ID,
				},
				ShortTitle:  insightGroup.ShortTitle,
				LongTitle:   insightGroup.LongTitle,
				Description: insightGroup.Description,
				LogoURL:     insightGroup.LogoURL,
				Insights:    insights,
			})
		}

		return nil
	})
}
