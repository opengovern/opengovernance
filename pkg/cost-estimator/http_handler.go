package cost_estimator

//type HttpHandler struct {
//}
//
//func InitializeHttpHandler(
//	postgresHost string, postgresPort string, postgresDb string, postgresUsername string, postgresPassword string, postgresSSLMode string,
//	logger *zap.Logger,
//) (h *HttpHandler, err error) {
//
//	fmt.Println("Initializing http handler")
//
//	cfg := postgres.Config{
//		Host:    postgresHost,
//		Port:    postgresPort,
//		User:    postgresUsername,
//		Passwd:  postgresPassword,
//		DB:      postgresDb,
//		SSLMode: postgresSSLMode,
//	}
//	orm, err := postgres.NewClient(&cfg, logger)
//	if err != nil {
//		return nil, fmt.Errorf("new postgres client: %w", err)
//	}
//	fmt.Println("Connected to the postgres database: ", postgresDb)
//
//	return &HttpHandler{
//		db: db,
//	}, nil
//}
