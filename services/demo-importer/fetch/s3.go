package fetch

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
)

func DownloadS3Object(url string) (string, error) {
	filename := path.Base(url)

	out, err := os.Create(filename)
	if err != nil {
		return "", err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", err
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	filePath := path.Join(cwd, filename)

	fmt.Printf("File downloaded successfully: %s\n", filePath)
	return filePath, nil
}
