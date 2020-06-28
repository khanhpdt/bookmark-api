package els

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
				"id": {
					"type": "keyword"
				},
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

// Search searches from the given index using the given body as search request.
func Search(index string, body io.Reader) (*SearchResult, error) {
	req := esapi.SearchRequest{
		Index: []string{index},
		Body:  body,
	}

	ctx, cancel := defaultContext()
	defer cancel()

	res, err := req.Do(ctx, es)

	if err != nil {
		log.Printf("Error searching from index %s: %s", index, err)
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, responseError(res)
	}

	var sResponse searchResponse
	if err := json.NewDecoder(res.Body).Decode(&sResponse); err != nil {
		return nil, fmt.Errorf("Error decoding search response: %s", err)
	}

	var searchResult = SearchResult{Total: sResponse.Hits.Total.Value, Hits: make([]*Hit, 0, len(sResponse.Hits.Hits))}
	for _, h := range sResponse.Hits.Hits {
		hit := Hit{ID: h.ID, Source: h.Source} // cannot reuse &h and must make a new struct here
		searchResult.Hits = append(searchResult.Hits, &hit)
	}

	return &searchResult, nil
}

func responseError(res *esapi.Response) error {
	var e map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
		return err
	}
	return fmt.Errorf("[%s] %s: %s", res.Status(), e["error"].(map[string]interface{})["type"], e["error"].(map[string]interface{})["reason"])
}

// Hit represents a search hit from ELS.
type Hit struct {
	ID     string          `json:"_id"`
	Source json.RawMessage `json:"_source"`
}

// SearchResult contains result of searching from ELS.
type SearchResult struct {
	Total int
	Hits  []*Hit
}

type searchResponse struct {
	Hits struct {
		Total struct {
			Value int
		}
		Hits []Hit
	}
}

func Delete(index, id string) error {
	req := esapi.DeleteRequest{Index: index, DocumentID: id}

	ctx, cancel := defaultContext()
	defer cancel()

	res, err := req.Do(ctx, es)

	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return responseError(res)
	}

	return nil
}
