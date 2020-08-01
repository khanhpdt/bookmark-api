package bookrepo

import (
	"encoding/json"
	"fmt"
	"github.com/khanhpdt/bookmark-api/internal/app/els"
	bookmodel "github.com/khanhpdt/bookmark-api/internal/app/model/book"
	"github.com/khanhpdt/bookmark-api/internal/app/mongo"
	"github.com/khanhpdt/bookmark-api/internal/app/repo/tagrepo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

// SaveUploadedBooks saves books to disk and database.
func SaveUploadedBooks(bookFiles []*multipart.FileHeader) []error {
	errs := make([]error, 0, len(bookFiles))

	for _, f := range bookFiles {
		fn := strings.ToLower(strings.ReplaceAll(f.Filename, " ", "_"))
		filePath := filepath.Join(getFileDir(), fn)

		if err := saveFileToDisk(f, filePath); err != nil {
			errs = append(errs, fmt.Errorf("error saving file %s to disk", f.Filename))
			continue
		}

		if err := saveBook(f.Filename, filePath); err != nil {
			errs = append(errs, fmt.Errorf("error saving book %s to database", f.Filename))
			continue
		}

		log.Printf("saved book %s to %s", f.Filename, filePath)
	}

	return errs
}

func getFileDir() string {
	homeDir, err := os.UserHomeDir()

	if err != nil {
		panic(fmt.Errorf("error getting user home directory: %s", err))
	}

	fileDir := filepath.Join(homeDir, "devbook-app", "books")

	if _, e := os.Stat(fileDir); os.IsNotExist(e) {
		err = os.MkdirAll(fileDir, os.ModePerm)
		log.Printf("Created directory for books at %s", fileDir)
	}

	if err != nil {
		panic(fmt.Errorf("error getting book directory at %s: %s", fileDir, err))
	}

	return fileDir
}

func closeFile(f multipart.File) {
	err := f.Close()
	if err != nil {
		log.Print("error closing file")
	}
}

func saveFileToDisk(f *multipart.FileHeader, filePath string) error {
	src, err := f.Open()
	if err != nil {
		return err
	}
	defer closeFile(src)

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer closeFile(out)

	_, err = io.Copy(out, src)
	if err != nil {
		return err
	}

	return nil
}

func saveBook(title, filePath string) error {
	ctx, cancelFunc := mongo.DefaultCtx()
	defer cancelFunc()

	id := primitive.NewObjectID()

	_, err := mongo.BookColl().InsertOne(ctx, bson.M{"_id": id, "title": title, "filePath": filePath})
	if err != nil {
		return fmt.Errorf("error saving doc to db: %s", err)
	}

	elsDoc := BookElsDoc{ID: id.Hex(), Title: title, FilePath: filePath}
	payload, err := json.Marshal(&elsDoc)
	if err != nil {
		return fmt.Errorf("error marshaling doc: %s", err)
	}

	if err := els.Index("book", id.Hex(), payload); err != nil {
		return fmt.Errorf("error indexing doc: %s", err)
	}

	return nil
}

// BookElsDoc represents book document in ELS.
type BookElsDoc struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	FilePath string   `json:"filePath"`
	Tags     []string `json:"tags"`
}

// BookSearchResult represents the result when searching books.
type BookSearchResult struct {
	List  []*BookElsDoc `json:"list"`
	Total int           `json:"total"`
}

// FindBooks search books from ELS using the given query.
func FindBooks(query io.Reader) (*BookSearchResult, error) {
	elsRes, err := els.Search("book", query)

	if err != nil {
		return nil, err
	}

	res := BookSearchResult{Total: elsRes.Total}

	docs, err := convertHitsToDocs(elsRes.Hits)
	if err != nil {
		return nil, err
	}
	res.List = docs

	return &res, nil
}

func convertHitsToDocs(hits []*els.Hit) ([]*BookElsDoc, error) {
	docs := make([]*BookElsDoc, 0, len(hits))
	for _, f := range hits {
		doc := new(BookElsDoc)
		if err := json.Unmarshal(f.Source, doc); err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

// FindByID finds doc from ELS with the given id.
func FindByID(id string) (*BookElsDoc, error) {
	query := fmt.Sprintf(`{ "query": { "ids": { "values": [ "%s" ] } } }`, id)
	elsRes, err := els.Search("book", strings.NewReader(query))
	if err != nil {
		return nil, err
	}

	docs, err := convertHitsToDocs(elsRes.Hits)
	if err != nil {
		return nil, err
	}

	if len(docs) == 0 {
		return nil, fmt.Errorf("book %s not found", id)
	}
	if len(docs) > 1 {
		return nil, fmt.Errorf("duplicated books for id %s", id)
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

	_, err = mongo.BookColl().DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return err
	}

	err = els.Delete("book", id)
	if err != nil {
		return err
	}

	return nil
}

func UpdateByID(id string, update bookmodel.UpdateRequest) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	ctx, cancelFunc := mongo.DefaultCtx()
	defer cancelFunc()

	currentBook, err := findMongoDoc(id)
	if err != nil {
		log.Printf("error reading book %s from mongo: %s", id, err)
		return err
	}

	query := bson.M{"_id": oid}
	updateObj := bson.M{"$set": bson.M{"title": update.Title, "tags": update.Tags}}
	_, err = mongo.BookColl().UpdateOne(ctx, query, updateObj)
	if err != nil {
		log.Printf("error saving book to mongo: %s", err)
		return err
	}

	err = reindex("book", id)
	if err != nil {
		log.Printf("error reindexing book: %s", err)
		return err
	}

	err = tagrepo.UpdateTags(currentBook.Tags, update.Tags)
	if err != nil {
		log.Printf("error updating tags for book %s: %s", id, err)
		return err
	}

	return nil
}

func reindex(indexName, id string) error {
	doc, err := findMongoDoc(id)
	if err != nil {
		return err
	}

	var elsDoc = BookElsDoc{
		ID:       doc.Id.Hex(),
		Title:    doc.Title,
		FilePath: doc.FilePath,
		Tags:     doc.Tags,
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

type BookMongoDoc struct {
	Id       primitive.ObjectID `bson:"_id"`
	Title    string             `bson:"title"`
	FilePath string             `bson:"filePath"`
	Tags     []string           `bson:"tags"`
}

func findMongoDoc(id string) (*BookMongoDoc, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	ctx, cancelFunc := mongo.DefaultCtx()
	defer cancelFunc()

	var doc BookMongoDoc
	err = mongo.BookColl().FindOne(ctx, bson.M{"_id": oid}).Decode(&doc)
	if err != nil {
		return nil, err
	}

	return &doc, nil
}

// GetBookFile returns a buffered reader to the book file
func GetBookFile(bookDoc *BookElsDoc) (*os.File, int64, error) {
	f, err := os.Open(bookDoc.FilePath)
	if err != nil {
		return nil, 0, err
	}

	finfo, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}

	return f, finfo.Size(), nil
}
