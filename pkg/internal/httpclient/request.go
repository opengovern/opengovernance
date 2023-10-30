package httpclient

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"kaytu-engine/pkg/internal/httpserver"
	"mime/multipart"
	"net/http"
	url2 "net/url"
	"strconv"
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/labstack/echo/v4"
)

type EchoError struct {
	Message string `json:"message"`
}
type Context struct {
	Ctx context.Context

	UserRole      api.Role
	UserID        string
	WorkspaceName string
	WorkspaceID   string

	MaxUsers       int64
	MaxConnections int64
	MaxResources   int64
}

func (ctx *Context) Request() *http.Request {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) SetRequest(r *http.Request) {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) SetResponse(r *echo.Response) {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) Response() *echo.Response {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) IsTLS() bool {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) IsWebSocket() bool {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) Scheme() string {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) RealIP() string {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) Path() string {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) SetPath(p string) {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) Param(name string) string {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) ParamNames() []string {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) SetParamNames(names ...string) {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) ParamValues() []string {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) SetParamValues(values ...string) {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) QueryParam(name string) string {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) QueryParams() url2.Values {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) QueryString() string {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) FormValue(name string) string {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) FormParams() (url2.Values, error) {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) FormFile(name string) (*multipart.FileHeader, error) {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) MultipartForm() (*multipart.Form, error) {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) Cookie(name string) (*http.Cookie, error) {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) SetCookie(cookie *http.Cookie) {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) Cookies() []*http.Cookie {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) Get(key string) interface{} {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) Set(key string, val interface{}) {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) Bind(i interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) Validate(i interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) Render(code int, name string, data interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) HTML(code int, html string) error {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) HTMLBlob(code int, b []byte) error {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) String(code int, s string) error {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) JSON(code int, i interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) JSONPretty(code int, i interface{}, indent string) error {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) JSONBlob(code int, b []byte) error {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) JSONP(code int, callback string, i interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) JSONPBlob(code int, callback string, b []byte) error {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) XML(code int, i interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) XMLPretty(code int, i interface{}, indent string) error {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) XMLBlob(code int, b []byte) error {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) Blob(code int, contentType string, b []byte) error {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) Stream(code int, contentType string, r io.Reader) error {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) File(file string) error {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) Attachment(file string, name string) error {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) Inline(file string, name string) error {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) NoContent(code int) error {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) Redirect(code int, url string) error {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) Error(err error) {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) Handler() echo.HandlerFunc {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) SetHandler(h echo.HandlerFunc) {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) Logger() echo.Logger {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) SetLogger(l echo.Logger) {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) Echo() *echo.Echo {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) Reset(r *http.Request, w http.ResponseWriter) {
	//TODO implement me
	panic("implement me")
}

func (ctx *Context) ToHeaders() map[string]string {
	return map[string]string{
		httpserver.XKaytuUserIDHeader:         ctx.UserID,
		httpserver.XKaytuUserRoleHeader:       string(ctx.UserRole),
		httpserver.XKaytuWorkspaceIDHeader:    ctx.WorkspaceID,
		httpserver.XKaytuWorkspaceNameHeader:  ctx.WorkspaceName,
		httpserver.XKaytuMaxUsersHeader:       fmt.Sprintf("%d", ctx.MaxUsers),
		httpserver.XKaytuMaxConnectionsHeader: fmt.Sprintf("%d", ctx.MaxConnections),
		httpserver.XKaytuMaxResourcesHeader:   fmt.Sprintf("%d", ctx.MaxResources),
	}
}

func (ctx *Context) GetWorkspaceName() string {
	return ctx.WorkspaceName
}

func (ctx *Context) GetWorkspaceID() string {
	return ctx.WorkspaceID
}

func FromEchoContext(c echo.Context) *Context {
	wsID := c.Request().Header.Get(httpserver.XKaytuWorkspaceIDHeader)
	name := c.Request().Header.Get(httpserver.XKaytuWorkspaceNameHeader)
	role := c.Request().Header.Get(httpserver.XKaytuUserRoleHeader)
	id := c.Request().Header.Get(httpserver.XKaytuUserIDHeader)
	maxUsers, _ := strconv.ParseInt(c.Request().Header.Get(httpserver.XKaytuMaxUsersHeader), 10, 64)
	maxConnections, _ := strconv.ParseInt(c.Request().Header.Get(httpserver.XKaytuMaxConnectionsHeader), 10, 64)
	maxResources, _ := strconv.ParseInt(c.Request().Header.Get(httpserver.XKaytuMaxResourcesHeader), 10, 64)
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
	req.Header.Set(echo.HeaderContentType, "application/json")
	req.Header.Set(echo.HeaderContentEncoding, "gzip")
	req.Header.Add(echo.HeaderAcceptEncoding, "gzip")

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

	body := res.Body
	if res.Header.Get("Content-Encoding") == "gzip" {
		body, err = gzip.NewReader(res.Body)
		if err != nil {
			return statusCode, fmt.Errorf("gzip new reader: %w", err)
		}
		defer body.Close()
	}

	statusCode = res.StatusCode
	if res.StatusCode != http.StatusOK {
		d, err := io.ReadAll(body)
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

	return statusCode, json.NewDecoder(body).Decode(v)
}
