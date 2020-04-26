package rest

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	fileApi "github.com/khanhpdt/bookmark-api/internal/app/rest/file"
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

	setupApis(r)

	r.Run(":8081") // listen and serve on 0.0.0.0:8081
}

func setupApis(r *gin.Engine) {
	fileApi.Setup(r)
}
