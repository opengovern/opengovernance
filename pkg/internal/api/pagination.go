package api

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// Page model
// @Description Please fill nextMarker with "" for the first request. After that fill it with last response of server.
// @Description e.g.:
// @Description {"nextMarker": "", "size": 10} --> Server
// @Description Server --> {"nextMarker": "MGT=", "size": 10}
// @Description {"nextMarker": "MGT=", "size": 10} --> Server

type Page struct {
	// fill it with empty for the first request
	NextMarker string `json:"nextMarker"`
	Size       int    `json:"size" minimum:"1" validate:"required,gte=1"`
}

func (p Page) GetIndex() (int, error) {
	if p.NextMarker != "" && len(p.NextMarker) > 0 {
		return MarkerToIdx(p.NextMarker)
	}
	return 0, nil
}

func (p Page) NextPage() (Page, error) {
	return NextPage(p)
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
