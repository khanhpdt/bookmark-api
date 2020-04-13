package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.MaxMultipartMemory = 8 << 20 // 8 MB (default is 32 MB)

	r.POST("/upload", func(c *gin.Context) {
		file, err := c.FormFile("file")

		if err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("Error receiving file: %s", err.Error()))
			return
		}

		log.Println()
		filePath := fmt.Sprintf("/tmp/%s", file.Filename)
		fmt.Printf("Saving file %s to %s", file.Filename, filePath)

		if err := c.SaveUploadedFile(file, filePath); err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("Error saving file to %s: %s", filePath, err.Error()))
			return
		}

		c.String(http.StatusOK, fmt.Sprintf("File %sfile uploaded.", file.Filename))
	})

	r.Run(":8081") // listen and serve on 0.0.0.0:8081
}
