package workspace

import (
	"context"
)

func (ts *testSuite) TestCreateAndFindHelmRelease() {
	workspace, err := ts.initWorkspace()
	ts.NoError(err)

	err = ts.server.createHelmRelease(context.Background(), workspace)
	ts.NoError(err)

	helmRelease, err := ts.server.findHelmRelease(context.Background(), workspace)
	ts.NoError(err)
	ts.NotNil(helmRelease)
}

func (ts *testSuite) TestCreateAndDeleteHelmRelease() {
	workspace, err := ts.initWorkspace()
	ts.NoError(err)

	err = ts.server.createHelmRelease(context.Background(), workspace)
	ts.NoError(err)

	err = ts.server.deleteHelmRelease(context.Background(), workspace)
	ts.NoError(err)
}

func (ts *testSuite) TestFindHelmRelease() {
	workspace, err := ts.initWorkspace()
	ts.NoError(err)

	helmRelease, err := ts.server.findHelmRelease(context.Background(), workspace)
	ts.NoError(err)
	ts.Nil(helmRelease)
}

func (ts *testSuite) TestCreateAndDeleteTargetNamespace() {
	workspace, err := ts.initWorkspace()
	ts.NoError(err)

	name := workspace.ID.String()
	err = ts.server.createTargetNamespace(context.Background(), name)
	ts.NoError(err)

	ns, err := ts.server.findTargetNamespace(context.Background(), name)
	ts.NoError(err)
	ts.Equal(name, ns.Name)

	err = ts.server.deleteTargetNamespace(context.Background(), name)
	ts.NoError(err)

	ns, err = ts.server.findTargetNamespace(context.Background(), name)
	ts.NoError(err)
	ts.Nil(ns)
}
