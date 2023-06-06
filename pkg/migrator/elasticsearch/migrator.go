package elasticsearch

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	elasticsearchv7 "github.com/elastic/go-elasticsearch/v7"
	"go.uber.org/zap"
	"io"
	"io/fs"
	"net/http"
	url2 "net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func Run(es elasticsearchv7.Config, logger *zap.Logger, esFolder string) error {
	for {
		err := HealthCheck(es, logger)
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

	var address string
	for _, ad := range es.Addresses {
		address = ad
	}

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
		fn := filepath.Base(fp)
		idx := strings.LastIndex(fn, ".")
		fne := fn[:idx]

		tp := "_index_template"
		if strings.HasSuffix(fne, "_component_template") {
			tp = "_component_template"
		}

		url, err := url2.Parse(fmt.Sprintf("https://%s/%s/%s", address, tp, fne))
		if err != nil {
			return err
		}

		f, err := os.Open(fp)
		if err != nil {
			return err
		}

		req, err := http.NewRequest("PUT", url.String(), f)
		if err != nil {
			return err
		}

		req.Header.Set("Content-type", "application/json")

		client := http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 10,
				TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			},
		}
		res, err := client.Do(req)
		if err != nil {
			return err
		}

		if res.StatusCode != http.StatusOK {
			b, err := io.ReadAll(res.Body)
			if err != nil {
				return err
			}

			logger.Error("failed to create template",
				zap.Int("statusCode", res.StatusCode),
				zap.String("body", string(b)),
			)
			return errors.New("failed to create template")
		}
	}

	return nil
}

func HealthCheck(es elasticsearchv7.Config, logger *zap.Logger) error {
	var address string
	for _, ad := range es.Addresses {
		address = ad
	}

	url, err := url2.Parse(fmt.Sprintf("https://%s/_cluster/health", address))
	if err != nil {
		return err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return err
	}

	client := http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 10,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		},
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		logger.Error("failed to get status",
			zap.Int("statusCode", res.StatusCode),
			zap.String("body", string(b)),
		)
		return errors.New("failed to get cluster health")
	}

	var js map[string]interface{}
	if err := json.Unmarshal(b, &js); err != nil {
		return err
	}

	if js["status"] != "green" && js["status"] != "yellow" {
		return errors.New("unhealthy")
	}

	return nil
}
