package api

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

type Page struct {
	NextMarker string `json:"nextMarker"`
	Size       int    `json:"size"`
}

func NextPage(page Page) (Page, error) {
	idx, err := MarkerToIdx(page.NextMarker)
	if err != nil {
		return Page{}, err
	}

	page.NextMarker = MarkerFromIdx(idx + page.Size)
	return page, nil
}

func MarkerFromIdx(idx int) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", idx)))
}

func MarkerToIdx(marker string) (int, error) {
	if len(strings.TrimSpace(marker)) == 0 {
		return 0, nil
	}

	b, err := base64.StdEncoding.DecodeString(marker)
	if err != nil {
		return 0, err
	}

	i, err := strconv.Atoi(string(b))
	if err != nil {
		return 0, err
	}

	return i, nil
}