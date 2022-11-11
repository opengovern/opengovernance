package config

type Redis struct {
	Address string
}

type ElasticSearch struct {
	Address  string
	Username string
	Password string
}

type Postgres struct {
	Host     string
	Port     string
	DB       string
	Username string
	Password string
	SSLMode  string
}

type KeibiService struct {
	BaseURL string
}

type HttpServer struct {
	Address string
}

type RabbitMQ struct {
	Service  string
	Username string
	Password string
}

type Vault struct {
	Address string
	Role    string
	Token   string
	CaPath  string
	UseTLS  bool
}

type Kafka struct {
	Addresses string
	Topic     string
}
