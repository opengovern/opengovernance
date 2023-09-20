package alerting_service

import (
	"fmt"
	"os"

	"github.com/kaytu-io/kaytu-engine/pkg/alerting"
)

func main() {
	if err := alerting.Command().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
