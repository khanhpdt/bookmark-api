package filerepo

import (
	"encoding/json"
	"fmt"
	"github.com/khanhpdt/bookmark-api/internal/app/els"
	filemodel "github.com/khanhpdt/bookmark-api/internal/app/model/file"
	"github.com/khanhpdt/bookmark-api/internal/app/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

// SaveUploadedFiles saves files to disk and database.
func SaveUploadedFiles(fs []*multipart.FileHeader) []error {
	errs := make([]error, 0, len(fs))

	for _, f := range fs {
		fn := strings.ToLower(strings.ReplaceAll(f.Filename, " ", "_"))
		filePath := filepath.Join(getFileDir(), fn)

		if err := saveFileToDisk(f, filePath); err != nil {
			errs = append(errs, fmt.Errorf("error saving file %s to disk", f.Filename))
			continue
		}

		if err := saveFileDocument(f.Filename, filePath); err != nil {
			errs = append(errs, fmt.Errorf("error saving file %s to database", f.Filename))
			continue
		}

		log.Printf("Saved file %s to %s.", f.Filename, filePath)
	}

	return errs
}

func getFileDir() string {
	homeDir, err := os.UserHomeDir()

	if err != nil {
		panic(fmt.Errorf("error getting user home directory: %s", err))
	}

	fileDir := filepath.Join(homeDir, "devbook-app", "files")

	if _, e := os.Stat(fileDir); os.IsNotExist(e) {
		err = os.MkdirAll(fileDir, os.ModePerm)
		log.Printf("Created file directory at %s", fileDir)
	}

	if err != nil {
		panic(fmt.Errorf("error getting file directory at %s: %s", fileDir, err))
	}

	return fileDir
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
	ctx, cancelFunc := mongo.DefaultCtx()
	defer cancelFunc()

	id := primitive.NewObjectID()

	_, err := mongo.FileColl().InsertOne(ctx, bson.M{"_id": id, "name": fileName, "path": filePath})
	if err != nil {
		return fmt.Errorf("Error saving doc to db: %s", err)
	}

	elsDoc := FileElsDoc{ID: id.Hex(), Name: fileName, Path: filePath}
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
	ID   string   `json:"id"`
	Name string   `json:"name"`
	Path string   `json:"path"`
	Tags []string `json:"tags"`
}

// FileSearchResult represents the result when searching files.
type FileSearchResult struct {
	List  []*FileElsDoc `json:"list"`
	Total int           `json:"total"`
}

// SearchFiles search files from ELS using the given query.
func SearchFiles(query io.Reader) (*FileSearchResult, error) {
	elsRes, err := els.Search("file", query)

	if err != nil {
		return nil, err
	}

	res := FileSearchResult{Total: elsRes.Total}

	docs, err := convertHitsToDocs(elsRes.Hits)
	if err != nil {
		return nil, err
	}
	res.List = docs

	return &res, nil
}

func convertHitsToDocs(hits []*els.Hit) ([]*FileElsDoc, error) {
	docs := make([]*FileElsDoc, 0, len(hits))
	for _, f := range hits {
		doc := new(FileElsDoc)
		if err := json.Unmarshal(f.Source, doc); err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

// FindByID finds doc from ELS with the given id.
func FindByID(id string) (*FileElsDoc, error) {
	query := fmt.Sprintf(`{ "query": { "ids": { "values": [ "%s" ] } } }`, id)
	elsRes, err := els.Search("file", strings.NewReader(query))
	if err != nil {
		return nil, err
	}

	docs, err := convertHitsToDocs(elsRes.Hits)
	if err != nil {
		return nil, err
	}

	if len(docs) == 0 {
		return nil, fmt.Errorf("file %s not found", id)
	}
	if len(docs) > 1 {
		return nil, fmt.Errorf("duplicated files for id %s", id)
	}

	return docs[0], nil
}

func DeleteByID(id string) error {
	ctx, cancelFunc := mongo.DefaultCtx()
	defer cancelFunc()

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = mongo.FileColl().DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return err
	}

	err = els.Delete("file", id)
	if err != nil {
		return err
	}

	return nil
}

func UpdateByID(id string, update filemodel.UpdateRequest) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	ctx, cancelFunc := mongo.DefaultCtx()
	defer cancelFunc()

	query := bson.M{"_id": oid}
	updateObj := bson.M{"$set": bson.M{"name": update.Name, "tags": update.Tags}}
	_, err = mongo.FileColl().UpdateOne(ctx, query, updateObj)
	if err != nil {
		log.Printf("error saving file to mongo: %s", err)
		return err
	}

	err = reindex("file", id)
	if err != nil {
		log.Printf("error reindexing file: %s", err)
		return err
	}

	return nil
}

func reindex(indexName, id string) error {
	doc, err := findMongoDoc(id)
	if err != nil {
		return err
	}

	var elsDoc = FileElsDoc{
		ID:   doc.Id.Hex(),
		Name: doc.Name,
		Path: doc.Path,
		Tags: doc.Tags,
	}

	elsDocBytes, err := json.Marshal(elsDoc)
	if err != nil {
		return err
	}

	err = els.Index(indexName, id, elsDocBytes)
	if err != nil {
		return err
	}

	return nil
}

type FileMongoDoc struct {
	Id   primitive.ObjectID `bson:"_id"`
	Name string             `bson:"name"`
	Path string             `bson:"path"`
	Tags []string           `bson:"tags"`
}

func findMongoDoc(id string) (*FileMongoDoc, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	ctx, cancelFunc := mongo.DefaultCtx()
	defer cancelFunc()

	var doc FileMongoDoc
	err = mongo.FileColl().FindOne(ctx, bson.M{"_id": oid}).Decode(&doc)
	if err != nil {
		return nil, err
	}

	return &doc, nil
}

// ReadFile returns a buffered reader to the file
func ReadFile(fileDoc *FileElsDoc) (*os.File, int64, error) {
	f, err := os.Open(fileDoc.Path)
	if err != nil {
		return nil, 0, err
	}

	finfo, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}

	return f, finfo.Size(), nil
}
