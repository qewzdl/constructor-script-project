package logger

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type Format string

const (
	FormatJSON    Format = "json"
	FormatConsole Format = "console"

	defaultServiceName = "constructor-script-backend"
	defaultEnvironment = "development"
)

type Config struct {
	Service          string
	Environment      string
	Version          string
	Level            zerolog.Level
	Format           Format
	Output           io.Writer
	outputFile       *os.File
	EnableCaller     bool
	EnableStackTrace bool
	AdditionalFields map[string]interface{}
}

var (
	Logger zerolog.Logger

	loggerValue       atomic.Value
	levelValue        atomic.Value
	configValue       atomic.Value
	stackTraceEnabled atomic.Bool

	ctxFieldsKey = contextFieldsKey{}
	ctxLoggerKey = contextLoggerKey{}
)

type contextFieldsKey struct{}
type contextLoggerKey struct{}

func init() {
	base := zerolog.New(io.Discard).With().Timestamp().Logger()
	loggerValue.Store(base)
	levelValue.Store(zerolog.InfoLevel)
	Logger = base
}

func Init() {
	cfg, cfgErr := ConfigFromEnv()
	if cfgErr != nil {
		fmt.Fprintf(os.Stderr, "logger: %v\n", cfgErr)
	}

	if err := InitWithConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "logger: %v\n", err)
	}
}

func ConfigFromEnv() (Config, error) {
	env := strings.TrimSpace(os.Getenv("ENVIRONMENT"))
	if env == "" {
		env = defaultEnvironment
	}

	service := strings.TrimSpace(os.Getenv("LOG_SERVICE"))
	if service == "" {
		service = defaultServiceName
	}

	version := strings.TrimSpace(os.Getenv("LOG_VERSION"))

	level, levelErr := resolveLogLevel(os.Getenv("LOG_LEVEL"))
	format, formatErr := resolveLogFormat(os.Getenv("LOG_FORMAT"), env)

	enableCaller := env != "production"
	if val, ok, err := lookupEnvBool("LOG_CALLER"); err != nil {
		formatErr = errors.Join(formatErr, err)
	} else if ok {
		enableCaller = val
	}

	enableStackTrace := env != "production"
	if val, ok, err := lookupEnvBool("LOG_STACKTRACE"); err != nil {
		formatErr = errors.Join(formatErr, err)
	} else if ok {
		enableStackTrace = val
	}

	cfg := Config{
		Service:          service,
		Environment:      env,
		Version:          version,
		Level:            level,
		Format:           format,
		EnableCaller:     enableCaller,
		EnableStackTrace: enableStackTrace,
	}

	var cfgErr error
	if levelErr != nil {
		cfgErr = errors.Join(cfgErr, levelErr)
	}
	if formatErr != nil {
		cfgErr = errors.Join(cfgErr, formatErr)
	}

	// If LOG_FILE is set, open the file for append and use it as output.
	if filePath := strings.TrimSpace(os.Getenv("LOG_FILE")); filePath != "" {
		f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			cfgErr = errors.Join(cfgErr, fmt.Errorf("open log file: %w", err))
		} else {
			cfg.outputFile = f
			teeStdout := true
			if val, ok, err := lookupEnvBool("LOG_FILE_TEE_STDOUT"); err != nil {
				cfgErr = errors.Join(cfgErr, err)
			} else if ok {
				teeStdout = val
			}
			if teeStdout {
				cfg.Output = io.MultiWriter(f, os.Stdout)
			} else {
				cfg.Output = f
			}
		}
	}

	return cfg, cfgErr
}

func InitWithConfig(cfg Config) error {
	cfgCopy := cloneConfig(cfg)
	if err := applyConfig(cfgCopy); err != nil {
		return err
	}

	configValue.Store(cfgCopy)
	return nil
}

func Level() zerolog.Level {
	if value := levelValue.Load(); value != nil {
		if level, ok := value.(zerolog.Level); ok {
			return level
		}
	}

	return zerolog.InfoLevel
}

func With(fields map[string]interface{}) zerolog.Logger {
	if len(fields) == 0 {
		return loadLogger()
	}

	return loadLogger().With().Fields(cloneFields(fields)).Logger()
}

func Info(msg string, fields map[string]interface{}) {
	logger := loadLogger()
	event := withFields(logger.Info(), fields)
	event.Msg(msg)
}

func Error(err error, msg string, fields map[string]interface{}) {
	logger := loadLogger()
	event := errorEvent(logger.Error(), err)
	event = withFields(event, fields)
	event.Msg(msg)
}

func Warn(msg string, fields map[string]interface{}) {
	logger := loadLogger()
	event := withFields(logger.Warn(), fields)
	event.Msg(msg)
}

func Debug(msg string, fields map[string]interface{}) {
	logger := loadLogger()
	event := withFields(logger.Debug(), fields)
	event.Msg(msg)
}

func Fatal(msg string, fields map[string]interface{}) {
	logger := loadLogger()
	event := withFields(logger.Fatal(), fields)
	event.Msg(msg)
}

func InfoContext(ctx context.Context, msg string, fields map[string]interface{}) {
	logger := FromContext(ctx)
	event := withFields(logger.Info(), mergeContextFields(ctx, fields))
	event.Msg(msg)
}

func ErrorContext(ctx context.Context, err error, msg string, fields map[string]interface{}) {
	logger := FromContext(ctx)
	event := errorEvent(logger.Error(), err)
	event = withFields(event, mergeContextFields(ctx, fields))
	event.Msg(msg)
}

func WarnContext(ctx context.Context, msg string, fields map[string]interface{}) {
	logger := FromContext(ctx)
	event := withFields(logger.Warn(), mergeContextFields(ctx, fields))
	event.Msg(msg)
}

func DebugContext(ctx context.Context, msg string, fields map[string]interface{}) {
	logger := FromContext(ctx)
	event := withFields(logger.Debug(), mergeContextFields(ctx, fields))
	event.Msg(msg)
}

func ContextWithFields(ctx context.Context, fields map[string]interface{}) context.Context {
	if len(fields) == 0 {
		return ctx
	}

	if ctx == nil {
		ctx = context.Background()
	}

	merged := mergeFields(rawFieldsFromContext(ctx), fields)
	if merged == nil {
		return ctx
	}

	baseLogger := FromContext(ctx)
	ctx = context.WithValue(ctx, ctxFieldsKey, merged)
	ctx = context.WithValue(ctx, ctxLoggerKey, baseLogger.With().Fields(cloneFields(fields)).Logger())
	return ctx
}

func ContextWithLogger(ctx context.Context, log zerolog.Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, ctxLoggerKey, log)
}

func FieldsFromContext(ctx context.Context) map[string]interface{} {
	return cloneFields(rawFieldsFromContext(ctx))
}

func FromContext(ctx context.Context) zerolog.Logger {
	if ctx == nil {
		return loadLogger()
	}

	if value := ctx.Value(ctxLoggerKey); value != nil {
		if ctxLogger, ok := value.(zerolog.Logger); ok {
			return ctxLogger
		}
	}

	if fields := rawFieldsFromContext(ctx); len(fields) > 0 {
		return loadLogger().With().Fields(fields).Logger()
	}

	return loadLogger()
}

func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery
		method := c.Request.Method
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()
		host := c.Request.Host

		requestFields := map[string]interface{}{
			"http_method": method,
			"http_path":   path,
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"host":        host,
		}

		if route := c.FullPath(); route != "" {
			requestFields["route"] = route
		}

		requestCtx := ContextWithFields(c.Request.Context(), requestFields)
		requestLogger := FromContext(requestCtx)
		c.Set("logger", requestLogger)
		c.Request = c.Request.WithContext(requestCtx)

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		pathWithQuery := path
		if rawQuery != "" {
			pathWithQuery = pathWithQuery + "?" + rawQuery
		}

		fields := map[string]interface{}{
			"client_ip":      clientIP,
			"method":         method,
			"path":           pathWithQuery,
			"status":         statusCode,
			"latency_ms":     latency.Milliseconds(),
			"user_agent":     userAgent,
			"host":           host,
			"response_bytes": c.Writer.Size(),
		}

		if route := c.FullPath(); route != "" {
			fields["route"] = route
		}

		if referer := c.Request.Referer(); referer != "" {
			fields["referer"] = referer
		}

		if c.Request.ContentLength > 0 {
			fields["content_length"] = c.Request.ContentLength
		}

		if requestID := requestIDFromGin(c); requestID != "" {
			fields["request_id"] = requestID
		}

		if errorMessage != "" {
			fields["error"] = errorMessage
		}

		logger := FromContext(c.Request.Context())

		switch {
		case statusCode >= http.StatusInternalServerError:
			logger.Error().Fields(fields).Msg("Server error")
		case statusCode >= http.StatusBadRequest:
			logger.Warn().Fields(fields).Msg("Client error")
		default:
			logger.Info().Fields(fields).Msg("Request completed")
		}
	}
}

func GinRecovery(withStack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				err := fmt.Errorf("%v", rec)
				fields := map[string]interface{}{
					"panic":  fmt.Sprintf("%v", rec),
					"path":   c.Request.URL.Path,
					"method": c.Request.Method,
					"host":   c.Request.Host,
				}

				if route := c.FullPath(); route != "" {
					fields["route"] = route
				}

				if requestID := requestIDFromGin(c); requestID != "" {
					fields["request_id"] = requestID
				}

				if withStack {
					fields["stack"] = string(debug.Stack())
				}

				ErrorContext(c.Request.Context(), err, "Recovered from panic", fields)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()

		c.Next()
	}
}

type GormLogger struct {
	SlowThreshold time.Duration
}

type GormLoggerOption func(*GormLogger)

func WithSlowThreshold(duration time.Duration) GormLoggerOption {
	return func(gl *GormLogger) {
		if duration > 0 {
			gl.SlowThreshold = duration
		}
	}
}

func NewGormLogger(opts ...GormLoggerOption) gormlogger.Interface {
	gl := &GormLogger{
		SlowThreshold: 200 * time.Millisecond,
	}

	for _, opt := range opts {
		opt(gl)
	}

	return gl
}

func (l *GormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return l
}

func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	logger := FromContext(ctx)
	logger.Info().Msgf(msg, data...)
}

func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	logger := FromContext(ctx)
	logger.Warn().Msgf(msg, data...)
}

func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	logger := FromContext(ctx)
	logger.Error().Msgf(msg, data...)
}

func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := map[string]interface{}{
		"duration_ms": elapsed.Milliseconds(),
		"sql":         sql,
	}

	if rows >= 0 {
		fields["rows"] = rows
	}

	logger := FromContext(ctx)

	switch {
	case err != nil && errors.Is(err, gorm.ErrRecordNotFound):
		logger.Debug().Fields(fields).Msg("Database record not found")
	case err != nil:
		errorEvent(logger.Error(), err).Fields(fields).Msg("Database query error")
	case l.SlowThreshold > 0 && elapsed > l.SlowThreshold:
		fields["threshold_ms"] = l.SlowThreshold.Milliseconds()
		logger.Warn().Fields(fields).Msg("Slow SQL query")
	default:
		logger.Debug().Fields(fields).Msg("Database query")
	}
}

func applyConfig(cfg Config) error {
	writer := cfg.Output
	if writer == nil {
		writer = os.Stdout
	}

	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.DurationFieldUnit = time.Millisecond
	zerolog.DurationFieldInteger = false

	var base zerolog.Logger
	switch cfg.Format {
	case FormatConsole:
		base = zerolog.New(newConsoleWriter(writer))
	default:
		base = zerolog.New(writer)
	}

	builder := base.With().Timestamp()
	if cfg.EnableCaller {
		builder = builder.Caller()
	}

	if cfg.Service != "" {
		builder = builder.Str("service", cfg.Service)
	}

	if cfg.Environment != "" {
		builder = builder.Str("environment", cfg.Environment)
	}

	if cfg.Version != "" {
		builder = builder.Str("version", cfg.Version)
	}

	if len(cfg.AdditionalFields) > 0 {
		builder = builder.Fields(cfg.AdditionalFields)
	}

	logger := builder.Logger().Level(cfg.Level)

	stackTraceEnabled.Store(cfg.EnableStackTrace)

	storeLogger(logger)
	zerolog.SetGlobalLevel(cfg.Level)
	levelValue.Store(cfg.Level)
	log.Logger = logger

	return nil
}

func cloneConfig(cfg Config) Config {
	clone := cfg
	if len(cfg.AdditionalFields) > 0 {
		clone.AdditionalFields = cloneFields(cfg.AdditionalFields)
	}
	return clone
}

func loadLogger() zerolog.Logger {
	if value := loggerValue.Load(); value != nil {
		if logger, ok := value.(zerolog.Logger); ok {
			return logger
		}
	}

	return Logger
}

func storeLogger(logger zerolog.Logger) {
	loggerValue.Store(logger)
	Logger = logger
}

func cloneFields(fields map[string]interface{}) map[string]interface{} {
	if len(fields) == 0 {
		return nil
	}

	cloned := make(map[string]interface{}, len(fields))
	for key, value := range fields {
		cloned[key] = value
	}

	return cloned
}

func mergeFields(base map[string]interface{}, additional map[string]interface{}) map[string]interface{} {
	if len(base) == 0 && len(additional) == 0 {
		return nil
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

func withFields(event *zerolog.Event, fields map[string]interface{}) *zerolog.Event {
	if len(fields) == 0 {
		return event
	}

	return event.Fields(fields)
}

func mergeContextFields(ctx context.Context, fields map[string]interface{}) map[string]interface{} {
	return mergeFields(FieldsFromContext(ctx), fields)
}

func rawFieldsFromContext(ctx context.Context) map[string]interface{} {
	if ctx == nil {
		return nil
	}

	if value := ctx.Value(ctxFieldsKey); value != nil {
		if fields, ok := value.(map[string]interface{}); ok {
			return fields
		}
	}

	return nil
}

func requestIDFromGin(c *gin.Context) string {
	if value, exists := c.Get("request_id"); exists {
		if id, ok := value.(string); ok && id != "" {
			return id
		}
	}

	if header := c.Request.Header.Get("X-Request-ID"); header != "" {
		return header
	}

	return ""
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

func resolveLogFormat(value string, environment string) (Format, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))

	switch trimmed {
	case "":
		return FormatConsole, nil
	case "json":
		return FormatJSON, nil
	case "console", "pretty":
		return FormatConsole, nil
	default:
		return FormatJSON, fmt.Errorf("invalid LOG_FORMAT %q", value)
	}
}

func lookupEnvBool(key string) (bool, bool, error) {
	raw, ok := os.LookupEnv(key)
	if !ok {
		return false, false, nil
	}

	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return false, true, fmt.Errorf("invalid boolean value %q for %s", raw, key)
	}

	value, err := strconv.ParseBool(trimmed)
	if err != nil {
		return false, true, fmt.Errorf("invalid boolean value %q for %s", raw, key)
	}

	return value, true, nil
}

func errorEvent(event *zerolog.Event, err error) *zerolog.Event {
	if err == nil {
		return event
	}

	if stackTraceEnabled.Load() {
		return event.Err(err).Str("stack", string(debug.Stack()))
	}

	return event.Err(err)
}

// Close closes any file used by the logger's output. It is safe to call
// multiple times. Standard outputs (stdout/stderr) will not be closed.
func Close() error {
	if v := configValue.Load(); v != nil {
		if cfg, ok := v.(Config); ok {
			if cfg.outputFile != nil {
				return cfg.outputFile.Close()
			}
			if cfg.Output != nil {
				if f, ok := cfg.Output.(*os.File); ok {
					if f == os.Stdout || f == os.Stderr {
						return nil
					}
					return f.Close()
				}
			}
		}
	}

	return nil
}

func newConsoleWriter(writer io.Writer) zerolog.ConsoleWriter {
	cw := zerolog.ConsoleWriter{
		Out:        writer,
		TimeFormat: "2006-01-02 15:04:05.000Z07:00",
		PartsOrder: []string{
			zerolog.TimestampFieldName,
			zerolog.LevelFieldName,
			zerolog.CallerFieldName,
			zerolog.MessageFieldName,
			"path",
		},
		FieldsOrder: []string{
			"route",
			"method",
			"status",
			"latency_ms",
			"duration_ms",
			"rows",
			"request_id",
			"host",
		},
		FieldsExclude: []string{"stack", "sql", "path"},
	}

	cw.FormatFieldName = func(i interface{}) string {
		if i == nil {
			return ""
		}
		return fmt.Sprintf("\n    %s=", i)
	}
	cw.FormatFieldValue = func(i interface{}) string {
		if i == nil {
			return ""
		}
		return fmt.Sprintf("%v", i)
	}
	cw.FormatErrFieldName = cw.FormatFieldName
	cw.FormatErrFieldValue = cw.FormatFieldValue

	cw.FormatPartValueByName = func(value interface{}, name string) string {
		if name == "path" {
			if value == nil {
				return ""
			}
			return fmt.Sprintf("\n    path=%v", value)
		}
		return fmt.Sprintf("%v", value)
	}

	cw.FormatExtra = func(fields map[string]interface{}, buf *bytes.Buffer) error {
		if sql, ok := fields["sql"]; ok {
			buf.WriteString("\n    sql: ")
			buf.WriteString(fmt.Sprint(sql))
		}
		if stack, ok := fields["stack"]; ok {
			buf.WriteString("\n    stack:\n")
			buf.WriteString(indentMultiline(fmt.Sprint(stack), "      "))
		}
		return nil
	}

	return cw
}

func indentMultiline(value string, indent string) string {
	value = strings.TrimRight(value, "\n")
	if value == "" {
		return ""
	}

	lines := strings.Split(value, "\n")
	for i, line := range lines {
		lines[i] = indent + line
	}

	return strings.Join(lines, "\n")
}
