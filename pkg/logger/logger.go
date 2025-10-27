package logger

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm/logger"
)

var Logger zerolog.Logger

type contextFieldsKey struct{}

var ctxFieldsKey contextFieldsKey

func Init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	level, levelErr := resolveLogLevel(os.Getenv("LOG_LEVEL"))
	var base zerolog.Logger

	if os.Getenv("ENVIRONMENT") == "development" {
		base = zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}).With().Timestamp().Caller().Logger()
	} else {

		base = zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()
	}

	Logger = base.Level(level)
	log.Logger = Logger

	if levelErr != nil {
		Logger.Warn().Err(levelErr).Str("level", os.Getenv("LOG_LEVEL")).Msg("Invalid LOG_LEVEL provided; falling back to info")
	}
}

func Info(msg string, fields map[string]interface{}) {
	event := withFields(Logger.Info(), fields)
	event.Msg(msg)
}

func Error(err error, msg string, fields map[string]interface{}) {
	event := Logger.Error()
	if err != nil {
		event = event.Err(err)
	}
	event = withFields(event, fields)
	event.Msg(msg)
}

func Warn(msg string, fields map[string]interface{}) {
	event := withFields(Logger.Warn(), fields)
	event.Msg(msg)
}

func Debug(msg string, fields map[string]interface{}) {
	event := withFields(Logger.Debug(), fields)
	event.Msg(msg)
}

func Fatal(msg string, fields map[string]interface{}) {
	event := withFields(Logger.Fatal(), fields)
	event.Msg(msg)
}

func InfoContext(ctx context.Context, msg string, fields map[string]interface{}) {
	event := withFields(Logger.Info(), mergeContextFields(ctx, fields))
	event.Msg(msg)
}

func ErrorContext(ctx context.Context, err error, msg string, fields map[string]interface{}) {
	event := Logger.Error()
	if err != nil {
		event = event.Err(err)
	}
	event = withFields(event, mergeContextFields(ctx, fields))
	event.Msg(msg)
}

func WarnContext(ctx context.Context, msg string, fields map[string]interface{}) {
	event := withFields(Logger.Warn(), mergeContextFields(ctx, fields))
	event.Msg(msg)
}

func DebugContext(ctx context.Context, msg string, fields map[string]interface{}) {
	event := withFields(Logger.Debug(), mergeContextFields(ctx, fields))
	event.Msg(msg)
}

func ContextWithFields(ctx context.Context, fields map[string]interface{}) context.Context {
	if len(fields) == 0 {
		return ctx
	}

	if ctx == nil {
		ctx = context.Background()
	}

	merged := mergeFields(FieldsFromContext(ctx), fields)
	if merged == nil {
		return ctx
	}

	return context.WithValue(ctx, ctxFieldsKey, merged)
}

func FieldsFromContext(ctx context.Context) map[string]interface{} {
	if ctx == nil {
		return nil
	}

	if fields, ok := ctx.Value(ctxFieldsKey).(map[string]interface{}); ok {
		copyFields := make(map[string]interface{}, len(fields))
		for key, value := range fields {
			copyFields[key] = value
		}
		return copyFields
	}

	return nil
}

func WithFields(fields map[string]interface{}) zerolog.Logger {
	if len(fields) == 0 {
		return Logger
	}

	return Logger.With().Fields(fields).Logger()
}

func withFields(event *zerolog.Event, fields map[string]interface{}) *zerolog.Event {
	if len(fields) == 0 {
		return event
	}

	return event.Fields(fields)
}

func mergeContextFields(ctx context.Context, fields map[string]interface{}) map[string]interface{} {
	return mergeFields(FieldsFromContext(ctx), fields)
}

func mergeFields(base map[string]interface{}, additional map[string]interface{}) map[string]interface{} {
	if len(base) == 0 && len(additional) == 0 {
		return nil
	}

	if len(base) == 0 {
		copyFields := make(map[string]interface{}, len(additional))
		for key, value := range additional {
			copyFields[key] = value
		}
		return copyFields
	}

	merged := make(map[string]interface{}, len(base)+len(additional))
	for key, value := range base {
		merged[key] = value
	}

	for key, value := range additional {
		merged[key] = value
	}

	return merged
}

func resolveLogLevel(value string) (zerolog.Level, error) {
	if strings.TrimSpace(value) == "" {
		return zerolog.InfoLevel, nil
	}

	level, err := zerolog.ParseLevel(strings.ToLower(value))
	if err != nil {
		return zerolog.InfoLevel, fmt.Errorf("parse log level: %w", err)
	}

	return level, nil
}

func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		method := c.Request.Method
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		requestCtx := ContextWithFields(c.Request.Context(), map[string]interface{}{
			"http_method": method,
			"http_path":   path,
			"client_ip":   clientIP,
			"user_agent":  userAgent,
		})
		c.Request = c.Request.WithContext(requestCtx)

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		pathWithQuery := path
		if raw != "" {
			pathWithQuery = pathWithQuery + "?" + raw
		}

		fields := map[string]interface{}{
			"client_ip":  clientIP,
			"method":     method,
			"path":       pathWithQuery,
			"status":     statusCode,
			"latency_ms": latency.Milliseconds(),
			"user_agent": userAgent,
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
