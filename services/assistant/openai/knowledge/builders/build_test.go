package builders

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/services/assistant/openai/knowledge/builders/tables"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"testing"
)

func TestBuildTablesFiles(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	logger = zap.NewNop()
	f, err := tables.ExtractTableFiles(logger)
	assert.NoError(t, err)

	for filename, content := range f {
		fmt.Println("===========", filename, "============")
		fmt.Println(content)
	}
}
