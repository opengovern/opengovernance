package workspace

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gitlab.com/keibiengine/keibi-engine/pkg/workspace/api"
)

func (ts *testSuite) initWorkspace() (*Workspace, error) {
	workspace := &Workspace{
		WorkspaceId: uuid.New(),
		Name:        ts.name,
		OwnerId:     ts.owner,
		Domain:      ts.name + ts.domainSuffix,
		Status:      StatusProvisioning.String(),
		Description: "workspace",
	}
	if err := ts.server.db.CreateWorkspace(workspace); err != nil {
		return nil, err
	}
	return workspace, nil
}

func (ts *testSuite) TestCreateWorkspace() {
	createWorkspaceTestCases := []struct {
		Workspace api.CreateWorkspaceRequest
		Owner     string
		Code      int
		Error     string
	}{
		{
			Workspace: api.CreateWorkspaceRequest{
				Name:        ts.name,
				Description: "workspace description",
			},
			Owner: ts.owner,
			Code:  http.StatusOK,
		},
		{
			Workspace: api.CreateWorkspaceRequest{
				Name:        ts.name,
				Description: "workspace description",
			},
			Owner: ts.owner,
			Code:  http.StatusFound,
			Error: "workspace already exists",
		},
		{
			Workspace: api.CreateWorkspaceRequest{
				Description: "workspace description",
			},
			Code:  http.StatusBadRequest,
			Error: "name is empty",
		},
		{
			Workspace: api.CreateWorkspaceRequest{
				Name:        ts.name,
				Description: "workspace description",
			},
			Code:  http.StatusUnauthorized,
			Error: "user id is empty",
		},
	}

	for i, tc := range createWorkspaceTestCases {
		tc := tc
		ts.T().Run(fmt.Sprintf("CreateWorkspaceTestCases-%d", i), func(t *testing.T) {
			data, _ := json.Marshal(tc.Workspace)

			r := httptest.NewRequest(http.MethodPost, "/api/v1/workspace", bytes.NewBuffer(data))
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			r.Header.Set(KeibiUserID, tc.Owner)
			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)
			err := ts.server.CreateWorkspace(c)
			if err != nil {
				var he *echo.HTTPError
				ts.Equal(true, errors.As(err, &he))
				ts.Equal(tc.Code, he.Code)

				if v, ok := he.Message.(error); ok {
					ts.Contains(v.Error(), tc.Error)
				} else {
					ts.Contains(he.Message, tc.Error)
				}
				return
			}

			var response api.CreateWorkspaceResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				ts.T().Fatalf("json decode: %v", err)
			}
			ts.NotEmpty(response.WorkspaceId)
		})
	}
}

func (ts *testSuite) TestDeleteWorkspace() {
	workspace, err := ts.initWorkspace()
	ts.NoError(err)

	deleteWorkspaceTestCases := []struct {
		WorkspaceId uuid.UUID
		Owner       string
		Code        int
		Error       string
	}{
		{
			WorkspaceId: workspace.WorkspaceId,
			Owner:       ts.owner,
			Code:        http.StatusOK,
		},
		{
			WorkspaceId: workspace.WorkspaceId,
			Code:        http.StatusUnauthorized,
			Error:       "user id is empty",
		},
		{
			WorkspaceId: uuid.UUID{},
			Owner:       ts.owner,
			Code:        http.StatusNotFound,
			Error:       "workspace not found",
		},
		{
			WorkspaceId: workspace.WorkspaceId,
			Owner:       "invalid owner",
			Code:        http.StatusForbidden,
			Error:       "operation is forbidden",
		},
	}

	for i, tc := range deleteWorkspaceTestCases {
		tc := tc
		ts.T().Run(fmt.Sprintf("DeleteWorkspaceTestCases-%d", i), func(t *testing.T) {
			r := httptest.NewRequest(http.MethodDelete, "/", nil)
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			r.Header.Set(KeibiUserID, tc.Owner)
			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)
			c.SetPath("/api/v1/workspace/:workspace_id")
			c.SetParamNames("workspace_id")
			c.SetParamValues(tc.WorkspaceId.String())

			err := ts.server.DeleteWorkspace(c)
			if err != nil {
				var he *echo.HTTPError
				ts.Equal(true, errors.As(err, &he))
				ts.Equal(tc.Code, he.Code)
				ts.Contains(he.Message, tc.Error)
				return
			}

			workspace, err := ts.server.db.GetWorkspace(tc.WorkspaceId)
			ts.NoError(err)
			ts.Equal(StatusDeleting.String(), workspace.Status)
		})
	}
}

func (ts *testSuite) TestListWorkspaces() {
	_, err := ts.initWorkspace()
	ts.NoError(err)

	listWorkspacesTestCases := []struct {
		Owner string
		Count int
		Code  int
		Error string
	}{
		{
			Code:  http.StatusUnauthorized,
			Error: "user id is empty",
		},
		{
			Owner: ts.owner,
			Code:  http.StatusOK,
			Count: 1,
		},
		{
			Owner: "invalid owner",
			Code:  http.StatusOK,
			Count: 0,
		},
	}

	for i, tc := range listWorkspacesTestCases {
		tc := tc
		ts.T().Run(fmt.Sprintf("ListWorkspacesTestCases-%d", i), func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces", nil)
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			r.Header.Set(KeibiUserID, tc.Owner)
			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)

			err := ts.server.ListWorkspaces(c)
			if err != nil {
				var he *echo.HTTPError
				ts.Equal(true, errors.As(err, &he))
				ts.Equal(tc.Code, he.Code)
				ts.Contains(he.Message, tc.Error)
				return
			}
		})
	}

}
