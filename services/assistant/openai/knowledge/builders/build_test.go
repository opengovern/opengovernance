package builders

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/services/assistant/openai/knowledge/builders/tables"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBuildTablesFiles(t *testing.T) {
	f, err := tables.ExtractTableFiles()
	assert.NoError(t, err)

	for filename, content := range f {
		fmt.Println("===========", filename, "============")
		fmt.Println(content)
	}
}
