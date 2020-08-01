package book

import (
	"bufio"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	bookmodel "github.com/khanhpdt/bookmark-api/internal/app/model/book"
	"github.com/khanhpdt/bookmark-api/internal/app/repo/bookrepo"
)

// Setup setups /books APIs.
func Setup(r *gin.Engine) {
	log.Printf("setting up /books APIs...")

	r.POST("/books/upload", uploadBook)
	r.POST("/books/search", findBooks)
	r.GET("/books/:bookID", findBookById)
	r.DELETE("/books/:bookID", deleteBookById)
	r.PUT("/books/:bookID", editBook)
	r.GET("/books/:bookID/download", downloadBook)
}

func uploadBook(c *gin.Context) {
	form, err := c.MultipartForm()

	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("error receiving books: %s", err.Error()))
		return
	}

	books := form.File["books"]
	if errs := bookrepo.SaveUploadedBooks(books); len(errs) > 0 {
		log.Printf("got %d errors when saving %d books", len(errs), len(books))
		for _, err := range errs {
			log.Print(err)
		}
		c.String(http.StatusBadRequest, fmt.Sprintf("error saving books"))
		return
	}

	c.String(http.StatusOK, fmt.Sprintf("uploaded %d books", len(books)))
}

func findBooks(c *gin.Context) {
	res, err := bookrepo.FindBooks(c.Request.Body)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, res)
}

func findBookById(c *gin.Context) {
	bookID := c.Param("bookID")
	book, err := bookrepo.FindByID(bookID)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, book)
}

func deleteBookById(c *gin.Context) {
	bookID := c.Param("bookID")

	err := bookrepo.DeleteByID(bookID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.String(http.StatusOK, "")
}

func editBook(c *gin.Context) {
	bookID := c.Param("bookID")

	var updateReq bookmodel.UpdateRequest
	if err := c.ShouldBindJSON(&updateReq); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	err := bookrepo.UpdateByID(bookID, updateReq)

	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.String(http.StatusOK, "")
}

func downloadBook(c *gin.Context) {
	bookID := c.Param("bookID")

	book, err := bookrepo.FindByID(bookID)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	osFile, size, err := bookrepo.GetBookFile(book)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	defer func() {
		if err = osFile.Close(); err != nil {
			log.Printf("error closing file %s", osFile.Name())
		}
	}()

	reader := bufio.NewReaderSize(osFile, 1000)

	c.DataFromReader(http.StatusOK, size, "application/pdf", reader, nil)
}
