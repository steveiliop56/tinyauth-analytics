package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ZerologMiddleware struct {
	level zerolog.Level
}

func NewZerologMiddleware(level zerolog.Level) *ZerologMiddleware {
	if level == zerolog.DebugLevel || level == zerolog.TraceLevel {
		log.Warn().Msg("Verbose log level is enabled. This may expose sensitive information in the logs.")
	}

	return &ZerologMiddleware{
		level: level,
	}
}

func (zm *ZerologMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tStart := time.Now()

		c.Next()

		code := c.Writer.Status()
		address := c.Request.RemoteAddr
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path

		latency := time.Since(tStart).String()

		subLogger := log.With().Str("method", method).
			Str("path", path).
			Int("status", code).
			Str("latency", latency).Logger()

		if zm.level == zerolog.DebugLevel {
			subLogger = subLogger.With().Str("address", address).Str("client_ip", clientIP).Logger()
		}

		switch {
		case code >= 400 && code < 500:
			subLogger.Warn().Msg("Client Error")
		case code >= 500:
			subLogger.Error().Msg("Server Error")
		default:
			subLogger.Info().Msg("Request")
		}
	}
}
