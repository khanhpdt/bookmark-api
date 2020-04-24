package els

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	elastic "github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
)

var es *elastic.Client

// Init initializes connection to ElasticSearch.
func Init() {
	cfg := elastic.Config{Addresses: []string{"http://localhost:9200"}}
	es7, err := elastic.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error connecting to ElasticSearch at %s: %s", cfg.Addresses, err)
	}

	log.Printf("Connected to ElasticSearch at %s.", cfg.Addresses)
	es = es7

	exist, err := checkIndexExist("file")
	if err != nil {
		log.Fatalf("Error when checking index exist: %s", err)
	}
	if !exist {
		log.Print("Creating index [file]...")
		createIndexFile()
	}
}

func createIndexFile() {
	body := `{
		"settings": {
			"analysis": {
				"normalizer": {
					"lowercase_normalizer": {
						"type": "custom",
						"char_filter": [],
						"filter": ["lowercase"]
					}
				}
			}
		},
		"mappings": {
			"properties": {
				"name": {
					"type":     "text",
					"analyzer": "standard"
				},
				"path": {
					"type":   		"keyword",
					"normalizer": "lowercase_normalizer"
				}
			}
		}
	}`

	req := esapi.IndicesCreateRequest{
		Index: "file",
		Body:  strings.NewReader(body),
	}

	ctx, cancel := defaultContext()
	defer cancel()

	res, err := req.Do(ctx, es)
	if err != nil {
		log.Fatalf("Error creating index [file]: %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		b := new(bytes.Buffer)
		b.ReadFrom(res.Body)
		log.Fatalf("Error creating index [file]: %s", b)
	}

	log.Print("Index [file] created.")
}

func defaultContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	return ctx, cancel
}

func checkIndexExist(index string) (bool, error) {
	req := esapi.IndicesExistsRequest{
		Index: []string{index},
	}

	ctx, cancel := defaultContext()
	defer cancel()

	res, err := req.Do(ctx, es)

	if err != nil {
		log.Printf("Error checking indices exist. %s", err)
		return false, err
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		log.Printf("Index '%s' found.", index)
		return true, nil
	} else if res.StatusCode == 404 {
		log.Printf("Index '%s' not found.", index)
		return false, nil
	} else {
		return false, fmt.Errorf("Status code %d not expected", res.StatusCode)
	}
}

// Index indexes the document to the index.
func Index(index, id string, body []byte) error {
	req := esapi.IndexRequest{
		Index:      index,
		DocumentID: id,
		Body:       bytes.NewReader(body),
	}

	ctx, cancel := defaultContext()
	defer cancel()

	res, err := req.Do(ctx, es)

	if err != nil {
		log.Printf("Error indexing document %s to index %s: %s", id, index, err)
		return err
	}
	defer res.Body.Close()

	return nil
}
