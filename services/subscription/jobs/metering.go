package jobs

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/services/subscription/service"
	"go.uber.org/zap"
	"time"
)

func GenerateMeters(svc service.MeteringService, logger *zap.Logger) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("metering paniced", zap.Error(fmt.Errorf("%v", r)))
			time.Sleep(5 * time.Second)
			go GenerateMeters(svc, logger)
		}
	}()

	for {
		logger.Info("starting checks")
		svc.RunChecks()
		time.Sleep(10 * time.Minute)
	}
}
