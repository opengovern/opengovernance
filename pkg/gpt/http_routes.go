package gpt

import (
	"fmt"
	authApi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	"github.com/labstack/echo/v4"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
)

func (h *HttpHandler) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	v1.POST("/gpt/run", httpserver.AuthorizeHandler(h.RunGPTQuery, authApi.ViewerRole))
}

// RunGPTQuery godoc
//
//	@Summary	Runs the query on KaytuGPT and returns the generated query
//	@Security	BearerToken
//	@Tags		resource
//	@Accept		json
//	@Produce	json
//	@Param		query	body		string	true	"Description of query for KaytuGPT"
//	@Success	200		{object}	map[string][]string
//	@Router		/ai/api/v1/gpt/run [post]
func (h *HttpHandler) RunGPTQuery(ctx echo.Context) error {
	var query string
	if err := ctx.Bind(&query); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	fileName := fmt.Sprintf("output-%d.json", rand.Int63())
	cmd := exec.Command("python3", "run.py", "-i", query, "-o", fileName)
	cmd.Dir = "/kaytu-ai"
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Println(string(out))
		return err
	}

	content, err := os.ReadFile(cmd.Dir + "/" + fileName)
	if err != nil {
		return err
	}

	return ctx.String(http.StatusOK, string(content))
}
