package logger

import (
	"context"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm/logger"
)

var Logger = logrus.New()

func Init() {
	Logger.SetOutput(os.Stdout)
	Logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05",
		ForceColors:     true,
		PadLevelText:    true,
	})
	Logger.SetLevel(logrus.DebugLevel)
}

func Info(msg string, fields map[string]interface{}) {
	Logger.WithFields(fields).Info(msg)
}

func Error(err error, msg string, fields map[string]interface{}) {
	Logger.WithError(err).WithFields(fields).Error(msg)
}

func Warn(msg string, fields map[string]interface{}) {
	Logger.WithFields(fields).Warn(msg)
}

func Debug(msg string, fields map[string]interface{}) {
	Logger.WithFields(fields).Debug(msg)
}

func Fatal(msg string, fields map[string]interface{}) {
	Logger.WithFields(fields).Fatal(msg)
}

func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()
		if raw != "" {
			path += "?" + raw
		}

		fields := logrus.Fields{
			"ip":     c.ClientIP(),
			"method": c.Request.Method,
			"path":   path,
			"status": status,
			"took":   duration,
		}

		switch {
		case status >= 500:
			Logger.WithFields(fields).Error("Server error")
		case status >= 400:
			Logger.WithFields(fields).Warn("Client error")
		default:
			Logger.WithFields(fields).Info("Request completed")
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
	Logger.Infof(msg, data...)
}

func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	Logger.Warnf(msg, data...)
}

func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	Logger.Errorf(msg, data...)
}

func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	if err != nil {
		Logger.WithError(err).WithFields(logrus.Fields{
			"sql":  sql,
			"rows": rows,
			"time": elapsed,
		}).Error("Database query error")
	} else if elapsed > l.SlowThreshold {
		Logger.WithFields(logrus.Fields{
			"sql":  sql,
			"rows": rows,
			"time": elapsed,
		}).Warn("Slow query")
	} else {
		Logger.WithFields(logrus.Fields{
			"sql":  sql,
			"rows": rows,
			"time": elapsed,
		}).Debug("Query executed")
	}
}
