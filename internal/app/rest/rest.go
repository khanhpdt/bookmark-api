package rest

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
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
	config.AllowOrigins = []string{"http://localhost:8080"} // for development
	r.Use(cors.New(config))

	r.MaxMultipartMemory = 8 << 20 // 8 MB (default is 32 MB)

	r.POST("/upload", func(c *gin.Context) {
		file, err := c.FormFile("file")

		if err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("Error receiving file: %s", err.Error()))
			return
		}

		filePath := fmt.Sprintf("/tmp/%s", file.Filename)

		if err := c.SaveUploadedFile(file, filePath); err != nil {
			errMsg := fmt.Sprintf("Error saving file to %s: %s", filePath, err.Error())
			log.Printf(errMsg)
			c.String(http.StatusBadRequest, errMsg)
			return
		}

		log.Printf("Saved file %s.", filePath)
		c.String(http.StatusOK, fmt.Sprintf("File %sfile uploaded.", file.Filename))
	})

	r.Run(":8081") // listen and serve on 0.0.0.0:8081
}
