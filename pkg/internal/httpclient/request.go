package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
)

type EchoError struct {
	Message string `json:"message"`
}
type Context struct {
	UserRole      api.Role
	UserID        string
	WorkspaceName string
}

func (ctx *Context) ToHeaders() map[string]string {
	return map[string]string{
		httpserver.XKeibiUserIDHeader:        ctx.UserID,
		httpserver.XKeibiUserRoleHeader:      string(ctx.UserRole),
		httpserver.XKeibiWorkspaceNameHeader: ctx.WorkspaceName,
	}
}

func FromEchoContext(c echo.Context) *Context {
	name := c.Request().Header.Get(httpserver.XKeibiWorkspaceNameHeader)
	role := c.Request().Header.Get(httpserver.XKeibiUserRoleHeader)
	id := c.Request().Header.Get(httpserver.XKeibiUserIDHeader)
	return &Context{
		WorkspaceName: name,
		UserRole:      api.Role(role),
		UserID:        id,
	}
}

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

		var echoerr EchoError
		if jserr := json.Unmarshal(d, &echoerr); jserr == nil {
			return fmt.Errorf(echoerr.Message)
		}

		return fmt.Errorf("http status: %d: %s", res.StatusCode, d)
	}
	if v == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(v)
}
