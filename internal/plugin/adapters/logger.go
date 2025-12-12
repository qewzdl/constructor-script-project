package adapters

import (
	"constructor-script-backend/pkg/logger"
	"constructor-script-backend/pkg/pluginsdk"
)

// LoggerAdapter adapts logger package to pluginsdk.Logger interface
type LoggerAdapter struct{}

func NewLoggerAdapter() pluginsdk.Logger {
	return &LoggerAdapter{}
}

func (a *LoggerAdapter) Debug(msg string, fields ...interface{}) {
	logger.Debug(msg, convertFieldsToMap(fields...))
}

func (a *LoggerAdapter) Info(msg string, fields ...interface{}) {
	logger.Info(msg, convertFieldsToMap(fields...))
}

func (a *LoggerAdapter) Warn(msg string, fields ...interface{}) {
	logger.Warn(msg, convertFieldsToMap(fields...))
}

func (a *LoggerAdapter) Error(msg string, fields ...interface{}) {
	logger.Error(nil, msg, convertFieldsToMap(fields...))
}

func (a *LoggerAdapter) Fatal(msg string, fields ...interface{}) {
	logger.Fatal(msg, convertFieldsToMap(fields...))
}

// convertFieldsToMap converts variadic interface{} to map[string]interface{}
// Expected format: key1, value1, key2, value2, ...
func convertFieldsToMap(fields ...interface{}) map[string]interface{} {
	if len(fields) == 0 {
		return nil
	}
	result := make(map[string]interface{})
	for i := 0; i < len(fields)-1; i += 2 {
		if key, ok := fields[i].(string); ok {
			result[key] = fields[i+1]
		}
	}
	return result
}
