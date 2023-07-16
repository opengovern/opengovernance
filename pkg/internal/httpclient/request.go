package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/labstack/echo/v4"

	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
)

type EchoError struct {
	Message string `json:"message"`
}
type Context struct {
	UserRole      api.Role
	UserID        string
	WorkspaceName string
	WorkspaceID   string

	MaxUsers       int64
	MaxConnections int64
	MaxResources   int64
}

func (ctx *Context) ToHeaders() map[string]string {
	return map[string]string{
		httpserver.XKeibiUserIDHeader:         ctx.UserID,
		httpserver.XKeibiUserRoleHeader:       string(ctx.UserRole),
		httpserver.XKeibiWorkspaceIDHeader:    ctx.WorkspaceID,
		httpserver.XKeibiWorkspaceNameHeader:  ctx.WorkspaceName,
		httpserver.XKeibiMaxUsersHeader:       fmt.Sprintf("%d", ctx.MaxUsers),
		httpserver.XKeibiMaxConnectionsHeader: fmt.Sprintf("%d", ctx.MaxConnections),
		httpserver.XKeibiMaxResourcesHeader:   fmt.Sprintf("%d", ctx.MaxResources),
	}
}

func (ctx *Context) GetWorkspaceName() string {
	return ctx.WorkspaceName
}

func (ctx *Context) GetWorkspaceID() string {
	return ctx.WorkspaceID
}

func FromEchoContext(c echo.Context) *Context {
	wsID := c.Request().Header.Get(httpserver.XKeibiWorkspaceIDHeader)
	name := c.Request().Header.Get(httpserver.XKeibiWorkspaceNameHeader)
	role := c.Request().Header.Get(httpserver.XKeibiUserRoleHeader)
	id := c.Request().Header.Get(httpserver.XKeibiUserIDHeader)
	maxUsers, _ := strconv.ParseInt(c.Request().Header.Get(httpserver.XKeibiMaxUsersHeader), 10, 64)
	maxConnections, _ := strconv.ParseInt(c.Request().Header.Get(httpserver.XKeibiMaxConnectionsHeader), 10, 64)
	maxResources, _ := strconv.ParseInt(c.Request().Header.Get(httpserver.XKeibiMaxResourcesHeader), 10, 64)
	return &Context{
		WorkspaceName:  name,
		WorkspaceID:    wsID,
		UserRole:       api.Role(role),
		UserID:         id,
		MaxUsers:       maxUsers,
		MaxResources:   maxResources,
		MaxConnections: maxConnections,
	}
}

func DoRequest(method string, url string, headers map[string]string, payload []byte, v interface{}) (statusCode int, err error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		return statusCode, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	for k, v := range headers {
		req.Header.Add(k, v)
	}
	t := http.DefaultTransport.(*http.Transport)
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100
	client := http.Client{
		Timeout:   3 * time.Minute,
		Transport: t,
	}
	res, err := client.Do(req)
	if err != nil {
		return statusCode, fmt.Errorf("do request: %w", err)
	}
	defer res.Body.Close()
	statusCode = res.StatusCode
	if res.StatusCode != http.StatusOK {
		d, err := io.ReadAll(res.Body)
		if err != nil {
			return statusCode, fmt.Errorf("read body: %w", err)
		}

		var echoerr EchoError
		if jserr := json.Unmarshal(d, &echoerr); jserr == nil {
			return statusCode, fmt.Errorf(echoerr.Message)
		}

		return statusCode, fmt.Errorf("http status: %d: %s", res.StatusCode, d)
	}
	if v == nil {
		return statusCode, nil
	}
	return statusCode, json.NewDecoder(res.Body).Decode(v)
}
