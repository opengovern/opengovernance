package main

import (
	"os"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe"
)

// @title Describe Scheduler Service
// @version 1.0
// @description Describe Scheduler Service
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8000
// @BasePath /
func main() {
	if err := describe.SchedulerCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
