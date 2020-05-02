package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kelseyhightower/envconfig"
	"github.com/p2pquake/web-realtime-api/server"
	"github.com/p2pquake/web-realtime-api/supplier"
)

type Config struct {
	APIKey          string `envconfig:"api_key" required:"true"`
	BindTo          string `envconfig:"bind_to" required:"true"`
	MongoURI        string `envconfig:"mongo_uri" required:"true"`
	MongoDatabase   string `envconfig:"mongo_database" required:"true"`
	MongoCollection string `envconfig:"mongo_collection" required:"true"`
}

func main() {
	var config Config
	err := envconfig.Process("", &config)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Starting...")
	ctx, cancel := context.WithCancel(context.Background())

	m := supplier.Mongo{}
	m.Start(ctx, config.MongoURI, config.MongoDatabase, config.MongoCollection)

	s := server.HTTP{}
	s.Start(ctx, config.BindTo)

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

L:
	for {
		select {
		case d := <-m.DataCh:
			json, err := json.Marshal(d)
			if err != nil {
				log.Printf("bson marshal error: %v\n", err)
			} else {
				log.Printf("broadcasting %s\n", string(json))
				s.Broadcast(string(json))
			}
		case <-quit:
			break L
		}
	}

	log.Println("Exiting...")
	cancel()
	<-s.Done
	log.Println("Bye!")
}
