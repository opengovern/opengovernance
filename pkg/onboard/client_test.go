package onboard

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestName(t *testing.T) {
	h := HttpHandler{
		masterAccessKey: "AKIAXMSWAQYFWO7WVLCK",
		masterSecretKey: "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",
	}
	cred := Credential{
		ID:                  uuid.UUID{},
		Name:                aws.String("o-y3x84b8zm6"),
		ConnectorType:       source.CloudAWS,
		Secret:              "AQICAHhNzpIghOTsfDBkWLoHl+9ZRYRT0FG2G6MV4j1Yu7S1cQElu7EyHHDE2EFdq95EHk/1AAAA+TCB9gYJKoZIhvcNAQcGoIHoMIHlAgEAMIHfBgkqhkiG9w0BBwEwHgYJYIZIAWUDBAEuMBEEDMzzYzCC+85R0R99ygIBEICBse/Yj2KiAE+D55136/u+brTmJBO48LGwZDdjfAvLXCUBmlTwC23YYL5C92bAAq50VACg4Mt2Lfi2T+yGaF3zQfBhMUFrvXVWa1eVAgxUsMeymlaewKSqu8vegPZAJaGlBCSOtI4uBch0KryFibyz7RrWt2Z6xxLjgvGX33SG9kOV4ZYMjBrJ56Io1LWLb8OwL+nZt6UHwdglqIOErRtbwaE9yalxzOYOCkbQ+FrMmvw6yg==",
		CredentialType:      "manual-aws-org",
		Enabled:             true,
		AutoOnboardEnabled:  true,
		LastHealthCheckTime: time.Time{},
		HealthStatus:        source.HealthStatusHealthy,
		HealthReason:        nil,
		Metadata:            nil,
		CreatedAt:           time.Time{},
		UpdatedAt:           time.Time{},
		DeletedAt:           sql.NullTime{},
		Version:             2,
	}
	onboardedSources, err := h.autoOnboardAWSAccountsV2(context.Background(), cred, 50)
	assert.NoError(t, err)
	fmt.Println(onboardedSources)
}
