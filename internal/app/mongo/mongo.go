package mongo

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var client *mongo.Client

const (
	mongoAddress = "localhost:27017"
	dbName       = "devbook"
)

// Init initializes connection to mongodb.
func Init() {
	log.Println("setting up mongodb connection...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c, err := mongo.Connect(ctx, options.Client().ApplyURI(fmt.Sprintf("mongodb://%s", mongoAddress)))
	if err != nil {
		panic(fmt.Sprintf("error connecting to mongodb at %s", mongoAddress))
	}

	client = c

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		panic(fmt.Sprintf("error checking connection to mongodb at %s", mongoAddress))
	}

	log.Printf("successfully connected to mongodb at %s", mongoAddress)
}

// BookColl returns the 'book' collection
func BookColl() *mongo.Collection {
	return client.Database(dbName).Collection("book")
}

func TagColl() *mongo.Collection {
	return client.Database(dbName).Collection("tag")
}

func DefaultCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
