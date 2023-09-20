package internal

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/kaytu-io/terraform-package/external/backend"
	azurem "github.com/kaytu-io/terraform-package/external/backend/remote-state/azure"
	"github.com/kaytu-io/terraform-package/external/backend/remote-state/s3"
	"github.com/kaytu-io/terraform-package/external/states"
	"github.com/kaytu-io/terraform-package/external/tfdiags"
	"io"
	"strings"

	"github.com/kaytu-io/terraform-package/external/states/statefile"
)

type Config struct {
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
}

func GetArns(content string) ([]string, error) {
	reader := io.Reader(strings.NewReader(content))
	arns := statefile.GetResourcesArn(reader)
	return arns, nil
}

func GetTypes(content string) ([]string, error) {
	reader := io.Reader(strings.NewReader(content))
	types := statefile.GetResourcesTypes(reader)
	return types, nil
}

func ParseAccountsFromArns(arns []string) ([]string, error) {
	haveAcc := make(map[string]bool)
	var accounts []string
	for _, arn := range arns {
		if arn[0] == '/' { // Azure
			splitArn := strings.Split(arn, "/")
			acc := splitArn[2]
			if _, value := haveAcc[acc]; !value && acc != "" {
				haveAcc[acc] = true
				accounts = append(accounts, acc)
			}
		} else { // AWS
			splitArn := strings.Split(arn, ":")
			acc := splitArn[4]
			if _, value := haveAcc[acc]; !value && acc != "" {
				haveAcc[acc] = true
				accounts = append(accounts, acc)
			}
		}
	}
	return accounts, nil
}

func GetResourceIDFromArn(arns []string) ([]string, error) {
	haveResource := make(map[string]bool)
	var resources []string
	for _, arn := range arns {
		if arn[0] == '/' { //Azure
			resources = append(resources, arn)
		} else { // AWS
			splitArn := strings.Split(arn, ":")
			res := strings.Split(splitArn[5], "/")[1]
			if _, value := haveResource[res]; !value && res != "" {
				haveResource[res] = true
				resources = append(resources, res)
			}
		}
	}
	return resources, nil
}

func GetRemoteState(config Config) *states.State {

	c := backend.TestWrapConfig(config.Config)

	var b backend.Backend
	if config.Type == "s3" {
		b = s3.New()
	} else if config.Type == "azurem" {
		b = azurem.New()
	}

	var diags tfdiags.Diagnostics

	// To make things easier for test authors, we'll allow a nil body here
	// (even though that's not normally valid) and just treat it as an empty
	// body.
	if c == nil {
		c = hcl.EmptyBody()
	}

	schema := b.ConfigSchema()
	spec := schema.DecoderSpec()
	obj, decDiags := hcldec.Decode(c, spec, nil)
	diags = diags.Append(decDiags)

	newObj, valDiags := b.PrepareConfig(obj)
	diags = diags.Append(valDiags.InConfigBody(c, ""))

	// it's valid for a Backend to have warnings (e.g. a Deprecation) as such we should only raise on errors
	if diags.HasErrors() {
		panic(diags.ErrWithWarnings())
	}

	obj = newObj

	confDiags := b.Configure(obj)
	if len(confDiags) != 0 {
		confDiags = confDiags.InConfigBody(c, "")
		panic(confDiags.ErrWithWarnings())
	}

	ws, err := b.Workspaces()
	if err != nil {
		panic(err)
	}

	stateMgr, err := b.StateMgr(ws[0])
	if err != nil {
		panic(err)
	}
	err = stateMgr.RefreshState()
	if err != nil {
		panic(err)
	}

	state := stateMgr.State()
	return state
}
