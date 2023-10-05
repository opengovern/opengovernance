package inventory

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func ExtractResourceCollections(baseDirectory string) ([]ResourceCollection, error) {
	var resourceCollections []ResourceCollection
	err := filepath.WalkDir(baseDirectory, func(path string, d fs.DirEntry, err error) error {
		if !strings.HasSuffix(path, ".json") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var resourceCollection ResourceCollection
		err = json.Unmarshal(content, &resourceCollection)
		if err != nil {
			return err
		}

		resourceCollections = append(resourceCollections, resourceCollection)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return resourceCollections, nil
}
