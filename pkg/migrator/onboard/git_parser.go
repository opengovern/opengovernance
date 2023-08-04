package onboard

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/kaytu-io/kaytu-engine/pkg/onboard"
)

type ConnectionGroup struct {
	Name  string `json:"name"`
	Query string `json:"query"`
}

type GitParser struct {
	connectionGroups []onboard.ConnectionGroup
}

func (g *GitParser) ExtractConnectionGroups(queryPath string) error {
	return filepath.WalkDir(queryPath, func(path string, d fs.DirEntry, err error) error {
		if strings.HasSuffix(path, ".json") {
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failure in reading file: %v", err)
			}

			var cg ConnectionGroup
			err = json.Unmarshal(content, &cg)
			if err != nil {
				return err
			}

			fileName := filepath.Base(path)
			if strings.HasSuffix(fileName, ".json") {
				fileName = fileName[:len(fileName)-len(".json")]
			}

			g.connectionGroups = append(g.connectionGroups, onboard.ConnectionGroup{
				Name:  fileName,
				Query: cg.Query,
			})
		}

		return nil
	})
}
