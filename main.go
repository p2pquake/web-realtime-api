package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

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

	log.Println("Starting...")
	ctx, cancel := context.WithCancel(context.Background())

	s := server.HTTP{}
	s.Start(ctx, config.BindTo)

	// wait terminate
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Exiting...")
	cancel()
	<-s.Done

	log.Println("Bye!")
}
