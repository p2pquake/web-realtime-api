package supplier

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Mongo struct {
	DataCh chan map[string]interface{}
	Done   chan struct{}

	context context.Context
}

func (m *Mongo) Start(context context.Context, uri string, database string, collection string) {
	m.DataCh = make(chan map[string]interface{}, 256)
	m.Done = make(chan struct{}, 1)
	m.context = context

	go m.run(uri, database, collection)
}

func (m *Mongo) run(uri string, db string, c string) {
	defer func() { m.Done <- struct{}{} }()

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
	filters := bson.D{{Key: "code", Value: bson.D{{Key: "$nin", Value: bson.A{5510, 5511}}}}}
	cur, err := col.Find(context.Background(), filters, &options)
	if err != nil {
		log.Fatal(err)
	}
	defer cur.Close(ctx)

	initial := true
	for {
		select {
		case <-m.context.Done():
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
				m.DataCh <- result
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
