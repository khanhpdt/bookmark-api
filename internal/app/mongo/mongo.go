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

const mongoAddress = "localhost:27017"

// Init initializes connection to mongodb.
func Init() {
	log.Println("Setting up mongodb connection...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c, err := mongo.Connect(ctx, options.Client().ApplyURI(fmt.Sprintf("mongodb://%s", mongoAddress)))
	if err != nil {
		panic(fmt.Sprintf("Error connecting to mongodb at %s", mongoAddress))
	}

	client = c

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		panic(fmt.Sprintf("Error checking connection to mongodb at %s", mongoAddress))
	}

	log.Printf("Successfully connected to mongodb at %s", mongoAddress)
}
