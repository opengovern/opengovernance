package httprequest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

func DoRequest(method, url string, headers map[string]string, payload []byte, v interface{}) error {
	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	client := http.Client{
		Timeout: 5 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		d, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("read body: %w", err)
		}
		return fmt.Errorf("http status: %d: %s", res.StatusCode, d)
	}
	if v == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(v)
}
