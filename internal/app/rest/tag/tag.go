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

	r.GET("/tags/suggestions", suggestTags)
}

func suggestTags(c *gin.Context) {
	res, err := tagrepo.SuggestTags()

	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, res)
}
