package tag

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"

	"github.com/khanhpdt/bookmark-api/internal/app/repo/tagrepo"
)

// Setup setups /tags APIs.
func Setup(r *gin.Engine) {
	log.Printf("Setting up /tags APIs...")

	r.GET("/tags", findTags)
}

func findTags(c *gin.Context) {
	res, err := tagrepo.FindTags()

	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, res)
}
