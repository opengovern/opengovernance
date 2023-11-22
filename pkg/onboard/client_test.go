package onboard

import (
	"fmt"
	apiv2 "github.com/kaytu-io/kaytu-engine/pkg/onboard/api/v2"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestName(t *testing.T) {
	h := HttpHandler{
		masterAccessKey: "AKIAXMSWAQYFWO7WVLCK",
		masterSecretKey: "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",
	}
	resp, err := h.createAWSCredential(apiv2.CreateCredentialRequest{
		Connector: source.CloudAWS,
		Config: apiv2.AWSCredentialConfig{
			AssumeRoleName:      "KaytuOrganizationCrossAccountRole",
			AccountID:           "517592840862",
			HealthCheckPolicies: nil,
			ExternalId:          nil,
		},
	})
	assert.NoError(t, err)
	fmt.Println(resp)
}
