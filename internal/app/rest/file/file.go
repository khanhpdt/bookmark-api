package file

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/khanhpdt/bookmark-api/internal/app/repo/filerepo"
)

// Setup setups /files APIs.
func Setup(r *gin.Engine) {
	log.Printf("Setting up /files APIs...")

	r.POST("/files/upload", uploadFile)
}

func uploadFile(c *gin.Context) {
	form, err := c.MultipartForm()

	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("Error receiving files: %s", err.Error()))
		return
	}

	files := form.File["files"]
	if errs := filerepo.SaveUploadedFiles(files); len(errs) > 0 {
		log.Printf("Got %d errors when saving %d files.", len(errs), len(files))
		for _, err := range errs {
			log.Print(err)
		}
		c.String(http.StatusBadRequest, fmt.Sprintf("Error saving files."))
		return
	}

	c.String(http.StatusOK, fmt.Sprintf("Uploaded %d files.", len(files)))
}
