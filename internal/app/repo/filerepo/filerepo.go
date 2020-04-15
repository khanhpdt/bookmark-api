package filerepo

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"strings"
	"time"

	"github.com/khanhpdt/bookmark-api/internal/app/mongo"
	"go.mongodb.org/mongo-driver/bson"
)

// SaveUploadedFile writes file to the given path.
func SaveUploadedFile(f *multipart.FileHeader) error {
	fn := strings.ToLower(strings.ReplaceAll(f.Filename, " ", "_"))
	filePath := fmt.Sprintf("/tmp/%s", fn)

	if err := saveFileToDisk(f, filePath); err != nil {
		return err
	}

	if err := saveFileDocument(f.Filename, filePath); err != nil {
		return err
	}

	return nil
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

	log.Printf("Saved file %s.", filePath)
	return nil
}

func saveFileDocument(fileName, filePath string) error {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	_, err := mongo.FileColl().InsertOne(ctx, bson.M{"name": fileName, "path": filePath})
	return err
}
