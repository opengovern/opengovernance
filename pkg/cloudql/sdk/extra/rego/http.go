package rego

import (
	"github.com/labstack/echo/v4"
	"github.com/open-policy-agent/opa/rego"
)

type RegoEvaluateRequest struct {
	Policies []string `json:"policies"`
	Query    string   `json:"query"`
}

type RegoEvaluateResponse struct {
	Results rego.ResultSet `json:"result"`
}

// evaluateEndpoint godoc
// @Summary Evaluate a rego policy
// @Description Evaluate a rego policy
// @Tags rego
// @Accept json
// @Produce json
// @Param request body RegoEvaluateRequest true "Rego Evaluate Request"
// @Success 200 {object} RegoEvaluateResponse
// @Router /evaluate [post]
func (r *RegoEngine) evaluateEndpoint(c echo.Context) error {
	req := new(RegoEvaluateRequest)
	if err := c.Bind(req); err != nil {
		return err
	}

	results, err := r.evaluate(c.Request().Context(), req.Policies, req.Query)
	if err != nil {
		return err
	}

	return c.JSON(200, RegoEvaluateResponse{Results: results})
}
