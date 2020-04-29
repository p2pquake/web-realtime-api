package main

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Supplier struct {
	DataCh chan bson.M
	Done   chan struct{}

	context context.Context
}

func (s *Supplier) Start(context context.Context, uri string, database string, collection string) {
	s.DataCh = make(chan bson.M, 256)
	s.Done = make(chan struct{}, 1)
	s.context = context

	go s.run(uri, database, collection)
}

func (s *Supplier) run(uri string, db string, c string) {
	defer func() { s.Done <- struct{}{} }()

	op := options.Client().ApplyURI(uri)
	client, err := mongo.NewClient(op)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	col := client.Database(db).Collection(c)

	ct := options.TailableAwait
	wait := time.Duration(1)
	options := options.FindOptions{CursorType: &ct, MaxAwaitTime: &wait}
	filter := bson.D{}

	var cur *mongo.Cursor
	defer cur.Close(ctx)

	cur, err = col.Find(ctx, filter, &options)
	if err != nil {
		log.Fatal(err)
	}
	defer cur.Close(ctx)

	initial := true
	for {
		select {
		case <-s.context.Done():
			log.Printf("Stopping tail due to context cancellation")
			break
		default:
		}

		if cur.TryNext(context.TODO()) {
			var result bson.M
			if err := cur.Decode(&result); err != nil {
				log.Fatal(err)
			}
			if !initial {
				s.DataCh <- result
			}
			continue
		} else {
			initial = false
		}

		if err := cur.Err(); err != nil {
			log.Fatal(err)
		}
		if cur.ID() == 0 {
			break
		}
	}
}
