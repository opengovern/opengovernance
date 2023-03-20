package utils

import (
	"net/url"
	"strings"
)

func ParseHTTPSubpathS3URIToBucketAndKey(uri string) (string, string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", "", err
	}

	hostParts := strings.Split(u.Hostname(), ".")

	bucket := hostParts[0]

	key := strings.TrimPrefix(u.Path, "/")

	return bucket, key, nil
}
