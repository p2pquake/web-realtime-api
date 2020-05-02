package main

import (
	"context"
	"log"

	"github.com/kelseyhightower/envconfig"
	"github.com/p2pquake/web-realtime-api/server"
)

type Config struct {
	APIKey string `envconfig:"api_key" required:"true"`
	BindTo string `envconfig:"bind_to" required:"true"`
}

func main() {
	var config Config
	err := envconfig.Process("", &config)
	if err != nil {
		log.Fatal(err)
	}

	s := server.HTTP{}
	s.Start(context.Background(), config.BindTo)
}
