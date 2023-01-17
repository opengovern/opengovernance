package describer

import (
	"context"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe/enums"
)

var (
	triggerTypeKey string = "trigger_type"
)

func WithTriggerType(ctx context.Context, tt enums.DescribeTriggerType) context.Context {
	return context.WithValue(ctx, triggerTypeKey, tt)
}

func GetTriggerTypeFromContext(ctx context.Context) enums.DescribeTriggerType {
	tt, ok := ctx.Value(triggerTypeKey).(enums.DescribeTriggerType)
	if !ok {
		return ""
	}
	return tt
}
