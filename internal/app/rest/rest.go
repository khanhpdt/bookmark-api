package rest

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/khanhpdt/bookmark-api/internal/app/repo/filerepo"
)

// Init initializes REST APIs.
func Init() {
	log.Println("Setting up REST APIs...")

	r := gin.Default()

	// - No origin allowed by default
	// - GET,POST, PUT, HEAD methods
	// - Credentials share disabled
	// - Preflight requests cached for 12 hours
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000"} // for development
	r.Use(cors.New(config))

	r.MaxMultipartMemory = 8 << 20 // 8 MB (default is 32 MB)

	r.POST("/upload", uploadFile)

	r.Run(":8081") // listen and serve on 0.0.0.0:8081
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
