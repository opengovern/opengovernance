package insight

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/kaytu-io/kaytu-util/pkg/model"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/db"
	"gorm.io/gorm"
)

type GitParser struct {
	insights      []db.Insight
	insightGroups []db.InsightGroup
}

func (g *GitParser) ExtractInsights(queryPath string) error {
	return filepath.WalkDir(queryPath, func(path string, d fs.DirEntry, err error) error {
		if strings.HasSuffix(path, ".json") {
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failure in reading file: %v", err)
			}

			var insight Insight
			err = json.Unmarshal(content, &insight)
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
				QueryID:     insight.QueryID,
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
		}

		return nil
	})
}

func (g *GitParser) ExtractInsightGroups(queryPath string) error {
	return filepath.WalkDir(queryPath, func(path string, d fs.DirEntry, err error) error {
		if strings.HasSuffix(path, ".json") {
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failure in reading file: %v", err)
			}

			var insightGroup InsightGroup
			err = json.Unmarshal(content, &insightGroup)
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
