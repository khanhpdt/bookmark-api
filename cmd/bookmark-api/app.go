package main

import (
	"log"

	"github.com/khanhpdt/bookmark-api/internal/app/els"
	"github.com/khanhpdt/bookmark-api/internal/app/mongo"
	"github.com/khanhpdt/bookmark-api/internal/app/rest"
)

func main() {
	log.Println("Starting application...")

	mongo.Init()
	els.Init()

	// call this last as it will block to listen to HTTP request
	rest.Init()
}
