package file

import (
	"bufio"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	filemodel "github.com/khanhpdt/bookmark-api/internal/app/model/file"
	"github.com/khanhpdt/bookmark-api/internal/app/repo/filerepo"
)

// Setup setups /files APIs.
func Setup(r *gin.Engine) {
	log.Printf("Setting up /files APIs...")

	r.POST("/files/upload", uploadFile)
	r.POST("/files/search", searchFiles)
	r.GET("/files/:fileID", findFile)
	r.DELETE("/files/:fileID", deleteFile)
	r.PUT("/files/:fileID", editFile)
	r.GET("/files/:fileID/download", downloadFile)
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

func searchFiles(c *gin.Context) {
	res, err := filerepo.SearchFiles(c.Request.Body)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, res)
}

func findFile(c *gin.Context) {
	fileID := c.Param("fileID")
	file, err := filerepo.FindByID(fileID)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, file)
}

func deleteFile(c *gin.Context) {
	fileID := c.Param("fileID")

	err := filerepo.DeleteByID(fileID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.String(http.StatusOK, "")
}

func editFile(c *gin.Context) {
	fileID := c.Param("fileID")

	var updateReq filemodel.UpdateRequest
	if err := c.ShouldBindJSON(&updateReq); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	err := filerepo.UpdateByID(fileID, updateReq)

	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.String(http.StatusOK, "")
}

func downloadFile(c *gin.Context) {
	fileID := c.Param("fileID")

	file, err := filerepo.FindByID(fileID)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	osFile, size, err := filerepo.ReadFile(file)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	defer osFile.Close()

	reader := bufio.NewReaderSize(osFile, 1000)

	c.DataFromReader(http.StatusOK, size, "application/pdf", reader, nil)
}
