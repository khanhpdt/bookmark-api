package filerepo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"strings"
	"time"

	"github.com/khanhpdt/bookmark-api/internal/app/els"
	"github.com/khanhpdt/bookmark-api/internal/app/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SaveUploadedFiles saves files to disk and database.
func SaveUploadedFiles(fs []*multipart.FileHeader) []error {
	errs := make([]error, 0, len(fs))

	for _, f := range fs {
		fn := strings.ToLower(strings.ReplaceAll(f.Filename, " ", "_"))
		filePath := fmt.Sprintf("/tmp/%s", fn)

		if err := saveFileToDisk(f, filePath); err != nil {
			errs = append(errs, fmt.Errorf("Error saving file %s to disk", f.Filename))
			continue
		}

		if err := saveFileDocument(f.Filename, filePath); err != nil {
			errs = append(errs, fmt.Errorf("Error saving file %s to database", f.Filename))
			continue
		}

		log.Printf("Saved file %s to %s.", f.Filename, filePath)
	}

	return errs
}

func saveFileToDisk(f *multipart.FileHeader, filePath string) error {
	src, err := f.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	if err != nil {
		return err
	}

	return nil
}

func saveFileDocument(fileName, filePath string) error {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()

	id := primitive.NewObjectID()

	_, err := mongo.FileColl().InsertOne(ctx, bson.M{"_id": id, "name": fileName, "path": filePath})
	if err != nil {
		return fmt.Errorf("Error saving doc to db: %s", err)
	}

	elsDoc := FileElsDoc{Name: fileName, Path: filePath}
	payload, err := json.Marshal(&elsDoc)
	if err != nil {
		return fmt.Errorf("Error marshaling doc: %s", err)
	}

	if err := els.Index("file", id.Hex(), payload); err != nil {
		return fmt.Errorf("Error indexing doc: %s", err)
	}

	return nil
}

// FileElsDoc represents file document in ELS.
type FileElsDoc struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// FileSearchResult represents the result when searching files.
type FileSearchResult struct {
	List  []FileSearchDoc `json:"list"`
	Total int             `json:"total"`
}

// FileSearchDoc represents a document in FileSearchResult.
type FileSearchDoc struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SearchFiles search files from ELS using the given query.
func SearchFiles(query []byte) (*FileSearchResult, error) {
	elsRes, err := els.Search("file", query)

	if err != nil {
		return nil, err
	}

	res := FileSearchResult{Total: elsRes.Total, List: make([]FileSearchDoc, 0, len(elsRes.Hits))}
	
	for _, f := range elsRes.Hits {
		file := FileSearchDoc{ID: f.ID}
		
		fileEls := new(FileElsDoc)
		if err := json.Unmarshal(f.Source, fileEls); err != nil {
			return nil, err
		}
		file.Name = fileEls.Name
		
		res.List = append(res.List, file)
	}

	return &res, nil
}
