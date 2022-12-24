// loads up the .env files to be used internally by Popcorn.

package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// uses go package: godotenv to load up development enviroment variables
func LoadDevConfig() {
	err := godotenv.Load("config/dev.env")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	}
}
