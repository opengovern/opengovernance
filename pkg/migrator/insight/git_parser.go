package insight

import (
	"encoding/json"
	"github.com/kaytu-io/kaytu-util/pkg/model"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/db"
	"gorm.io/gorm"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type GitParser struct {
	insights []db.Insight
}

func (g *GitParser) ExtractInsights(queryPath string) error {
	return filepath.WalkDir(queryPath, func(path string, d fs.DirEntry, err error) error {
		if strings.HasSuffix(path, ".json") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
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
