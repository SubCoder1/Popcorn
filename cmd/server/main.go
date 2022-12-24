// The name of the file says it all

package main

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// Version indicates the current version of the application.
var Version = "1.0.0"

func init() {
	currentDateTime := time.Now()
	fmt.Println("Welcome to Popcorn!")
	fmt.Printf("%d-%d-%d %d:%d:%d",
		currentDateTime.Day(),
		currentDateTime.Month(),
		currentDateTime.Year(),
		currentDateTime.Hour(),
		currentDateTime.Minute(),
		currentDateTime.Second(),
	)
}

func main() {
	server := gin.Default()

	Router(server)

	server.Run("localhost:8000")
}
