// The main file of Popcorn.

package main

import (
	"Popcorn/internal/config"
	"Popcorn/pkg/logger"
	"flag"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slices"
)

// Indicates the current version of Popcorn.
var Version = "1.0.0"

// Indicates the current environment of Popcorn.
var Env string

func init() {
	// Fetching the mandatory env argument passed with main.go.
	// values should be either DEV or PROD for development or production environment.
	flag.StringVar(&Env, "env", "", "Popcorn Environment: DEV, PROD")
	flag.Parse()
	if !slices.Contains([]string{"DEV", "PROD"}, Env) {
		flag.PrintDefaults()
		os.Exit(2)
	}
	// Setup logger with Prettified logs in DEV, JSON in PROD environment.
	logger.Setup(Env)
	logger.Logger.Info().Msg(fmt.Sprintf("Welcome to Popcorn v%s", Version))
}

func main() {
	var addr, port string
	if Env == "DEV" {
		// load the development configurations.
		config.LoadDevConfig()
		// Fetching addr and port depending upon env flag.
		addr, port = os.Getenv("ADDR"), os.Getenv("PORT")
		// This is the preferred mode used by gin server in DEV environment.
		gin.SetMode(gin.DebugMode)
	}
	// Initializing the gin server.
	server := gin.New()
	// Forcing gin to use custom Logger instead of the default one.
	server.Use(logger.LoggerGinExtension(&logger.Logger))
	server.Use(gin.Recovery())
	// Running router.Router() which routes all of the REST API groups and paths.
	Router(server)
	// Finally, run the gin server.
	server.Run(addr + ":" + port)
}
