package inventory

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"os"
)

type Option struct {
	Host string
	Port int
	User string
	Pass string
	Db   string
}

type SteampipeDatabase struct {
	db *pgxpool.Pool
}

func NewSteampipeDatabase(option Option, maxConnections int) *SteampipeDatabase {
	var err error

	if maxConnections == 0 {
		maxConnections = 5
	}

	config, _ := pgxpool.ParseConfig("")
	config.ConnConfig.Host = option.Host
	config.ConnConfig.Port = uint16(option.Port)
	config.ConnConfig.Database = option.Db
	config.ConnConfig.User = option.User
	config.ConnConfig.Password = option.Pass
	config.MaxConns = int32(maxConnections)

	fmt.Printf("Creating pgx connection pool. host: %v, port: %v\n", option.Host, option.Port)
	postgresPool, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		fmt.Printf("Unable to create connection pool. host: %v, error: %v\n", option.Host, err)
		os.Exit(1)
	}
	fmt.Println("Pgx connection pool created successfully.")

	return &SteampipeDatabase{postgresPool}
}

