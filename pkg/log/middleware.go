// This middleware is used to integrate zerolog extension created in logger.go into gin server.

package log

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// This middleware is replicated from https://learninggolang.com/it5-gin-structured-logging.html.
// Primary use-case of this middleware is to force gin to use zerolog functionality instead of the default one.
func LoggerGinExtension(logger Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now() // Start timer
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Fill the params
		param := gin.LogFormatterParams{}

		param.TimeStamp = time.Now() // Stop timer
		param.Latency = param.TimeStamp.Sub(start)
		if param.Latency > time.Minute {
			param.Latency = param.Latency.Truncate(time.Second)
		}

		param.ClientIP = c.ClientIP()
		param.Method = c.Request.Method
		param.StatusCode = c.Writer.Status()
		param.ErrorMessage = c.Errors.ByType(gin.ErrorTypePrivate).String()
		param.BodySize = c.Writer.Size()
		if raw != "" {
			path = path + "?" + raw
		}
		param.Path = path

		message := fmt.Sprintf("%s | %s | %s | %d | %s | %s",
			param.ClientIP,
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency.String(),
			param.ErrorMessage)

		if c.Writer.Status() >= 500 {
			logger.Error().Msg(message)
		} else if c.Writer.Status() >= 400 {
			logger.Warn().Msg(message)
		} else {
			logger.Info().Msg(message)
		}
	}
}
