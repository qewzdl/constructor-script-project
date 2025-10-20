package logger

import (
	"context"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm/logger"
)

var Logger zerolog.Logger

func Init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	if os.Getenv("ENVIRONMENT") == "development" {
		Logger = zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}).With().Timestamp().Caller().Logger()
	} else {

		Logger = zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()
	}

	log.Logger = Logger
}

func Info(msg string, fields map[string]interface{}) {
	event := Logger.Info()
	if fields != nil {
		event = event.Fields(fields)
	}
	event.Msg(msg)
}

func Error(err error, msg string, fields map[string]interface{}) {
	event := Logger.Error().Err(err)
	if fields != nil {
		event = event.Fields(fields)
	}
	event.Msg(msg)
}

func Warn(msg string, fields map[string]interface{}) {
	event := Logger.Warn()
	if fields != nil {
		event = event.Fields(fields)
	}
	event.Msg(msg)
}

func Debug(msg string, fields map[string]interface{}) {
	event := Logger.Debug()
	if fields != nil {
		event = event.Fields(fields)
	}
	event.Msg(msg)
}

func Fatal(msg string, fields map[string]interface{}) {
	event := Logger.Fatal()
	if fields != nil {
		event = event.Fields(fields)
	}
	event.Msg(msg)
}

func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		if raw != "" {
			path = path + "?" + raw
		}

		fields := map[string]interface{}{
			"client_ip":  clientIP,
			"method":     method,
			"path":       path,
			"status":     statusCode,
			"latency_ms": latency.Milliseconds(),
			"user_agent": c.Request.UserAgent(),
		}

		if requestID, exists := c.Get("request_id"); exists {
			if id, ok := requestID.(string); ok && id != "" {
				fields["request_id"] = id
			}
		}

		if errorMessage != "" {
			fields["error"] = errorMessage
		}

		if statusCode >= 500 {
			Logger.Error().Fields(fields).Msg("Server error")
		} else if statusCode >= 400 {
			Logger.Warn().Fields(fields).Msg("Client error")
		} else {
			Logger.Info().Fields(fields).Msg("Request completed")
		}
	}
}

type GormLogger struct {
	SlowThreshold time.Duration
}

func NewGormLogger() logger.Interface {
	return &GormLogger{
		SlowThreshold: 200 * time.Millisecond,
	}
}

func (l *GormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return l
}

func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	Logger.Info().Msgf(msg, data...)
}

func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	Logger.Warn().Msgf(msg, data...)
}

func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	Logger.Error().Msgf(msg, data...)
}

func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := map[string]interface{}{
		"duration_ms": elapsed.Milliseconds(),
		"rows":        rows,
		"sql":         sql,
	}

	if err != nil {
		Logger.Error().Err(err).Fields(fields).Msg("Database query error")
	} else if elapsed > l.SlowThreshold {
		Logger.Warn().Fields(fields).Msg("Slow SQL query")
	} else {
		Logger.Debug().Fields(fields).Msg("Database query")
	}
}
