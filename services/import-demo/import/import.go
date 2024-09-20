package _import

import (
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/labstack/gommon/log"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	es, err := elasticsearch.NewDefaultClient()
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}

	dir := "path/to/json/files"

	indexConfigs, err := ReadIndexConfigs(dir)
	if err != nil {
		log.Fatalf("Error reading index configs: %s", err)
	}

	for indexName, config := range indexConfigs {
		err := CreateIndex(es, indexName, config.Settings, config.Mappings)
		if err != nil {
			log.Fatalf("Error creating index %s: %s", indexName, err)
		}
	}

	dataFiles, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		log.Fatalf("Error reading data files: %s", err)
	}

	for _, file := range dataFiles {
		if strings.HasSuffix(file, ".mapping.json") || strings.HasSuffix(file, ".settings.json") {
			continue
		}

		indexName := strings.TrimSuffix(filepath.Base(file), ".json")
		if _, exists := indexConfigs[indexName]; exists {
			go func(filePath string, indexName string) {
				err := ProcessJSONFile(es, filePath, indexName)
				if err != nil {
					log.Printf("Error processing file %s for index %s: %s", filePath, indexName, err)
				}
			}(file, indexName)
		} else {
			log.Printf("No index config found for file: %s", file)
		}
	}

	time.Sleep(1 * time.Hour)
}
