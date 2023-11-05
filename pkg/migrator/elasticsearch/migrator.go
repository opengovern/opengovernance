package elasticsearch

import (
	"context"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"go.uber.org/zap"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func Run(es kaytu.Client, logger *zap.Logger, esFolder string) error {
	for {
		err := es.Healthcheck(context.TODO())
		if err != nil {
			if err.Error() == "unhealthy" {
				logger.Warn("Waiting for status to be GREEN or YELLOW. Sleeping for 10 seconds...")
				time.Sleep(10 * time.Second)
				continue
			}
			return err
		}
		break
	}
	logger.Warn("Starting es migration")

	var files []string
	err := filepath.Walk(esFolder, func(path string, info fs.FileInfo, err error) error {
		if strings.HasSuffix(info.Name(), ".json") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	for _, fp := range files {
		err = CreateTemplate(es, logger, fp)
		if err != nil {
			logger.Error("failed to create template", zap.Error(err), zap.String("filepath", fp))
		}
	}

	return nil
}

func CreateTemplate(es kaytu.Client, logger *zap.Logger, fp string) error {
	fn := filepath.Base(fp)
	idx := strings.LastIndex(fn, ".")
	fne := fn[:idx]

	f, err := os.ReadFile(fp)
	if err != nil {
		return err
	}

	if strings.HasSuffix(fne, "_component_template") {
		err = es.CreateComponentTemplate(context.TODO(), fne, string(f))
		if err != nil {
			logger.Error("failed to create component template", zap.Error(err), zap.String("filepath", fp))
			return err
		}
	} else {
		err = es.CreateIndexTemplate(context.TODO(), fne, string(f))
		if err != nil {
			logger.Error("failed to create index template", zap.Error(err), zap.String("filepath", fp))
			return err
		}
	}

	return nil
}
