package api

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

type PageResponse struct {
	NextMarker string `json:"nextMarker"`
	Size       int    `json:"size"`
	TotalCount int64  `json:"totalCount,omitempty"`
}

func (p PageResponse) GetIndex() (int, error) {
	if p.NextMarker != "" && len(p.NextMarker) > 0 {
		return MarkerToIdx(p.NextMarker)
	}
	return 0, nil
}

func (p PageResponse) NextPage() (PageResponse, error) {
	idx, err := MarkerToIdx(p.NextMarker)
	if err != nil {
		return PageResponse{}, err
	}

	p.NextMarker = MarkerFromIdx(idx + p.Size)
	return p, nil
}

func (p PageResponse) ToRequest() PageRequest {
	return PageRequest{
		NextMarker: p.NextMarker,
		Size:       p.Size,
	}
}

// PageRequest model
// @Description Please fill nextMarker with "" for the first request. After that fill it with last response of server.
// @Description e.g.:
// @Description {"nextMarker": "", "size": 10} --> Server
// @Description Server --> {"nextMarker": "MGT=", "size": 10}
// @Description {"nextMarker": "MGT=", "size": 10} --> Server
type PageRequest struct {
	// fill it with empty for the first request
	NextMarker string `json:"nextMarker"`
	Size       int    `json:"size" minimum:"1" validate:"required,gte=1"`
}

func (p PageRequest) GetIndex() (int, error) {
	if p.NextMarker != "" && len(p.NextMarker) > 0 {
		return MarkerToIdx(p.NextMarker)
	}
	return 0, nil
}

func (p PageRequest) NextPage() (PageRequest, error) {
	idx, err := MarkerToIdx(p.NextMarker)
	if err != nil {
		return PageRequest{}, err
	}

	p.NextMarker = MarkerFromIdx(idx + p.Size)
	return p, nil
}

func (p PageRequest) ToResponse(totalCount int64) PageResponse {
	return PageResponse{
		NextMarker: p.NextMarker,
		Size:       p.Size,
		TotalCount: totalCount,
	}
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
