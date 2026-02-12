package log

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ log.Logger = (*ZapLogger)(nil)

// ZapLogger is a logger impl.
type ZapLogger struct {
	log  *zap.Logger
	Sync func() error
}

// NewZapLogger return a zap logger.
func NewZapLogger(encoder zapcore.Encoder, level zap.AtomicLevel, opts ...zap.Option) *ZapLogger {
	core := zapcore.NewCore(
		encoder,
		zapcore.NewMultiWriteSyncer(
			zapcore.AddSync(os.Stdout),
		), level)
	zapLogger := zap.New(core, opts...)
	return &ZapLogger{log: zapLogger, Sync: zapLogger.Sync}
}

// Log Implementation of logger interface.
func (l *ZapLogger) Log(level log.Level, keyvals ...interface{}) error {
	if len(keyvals) == 0 || len(keyvals)%2 != 0 {
		l.log.Warn(fmt.Sprint("Keyvalues must appear in pairs: ", keyvals))
		return nil
	}
	// Add caller as the first field
	data := []zap.Field{zap.String("caller", getCaller())}
	// Zap.Field is used when keyvals pairs appear
	for i := 0; i < len(keyvals); i += 2 {
		data = append(data, zap.Any(fmt.Sprint(keyvals[i]), fmt.Sprint(keyvals[i+1])))
	}
	switch level {
	case log.LevelDebug:
		l.log.Debug("", data...)
	case log.LevelInfo:
		l.log.Info("", data...)
	case log.LevelWarn:
		l.log.Warn("", data...)
	case log.LevelError:
		l.log.Error("", data...)
	case log.LevelFatal:
		l.log.Fatal("", data...)
	}
	return nil
}

// InitDefaultLogger creates a console logger.
func InitDefaultLogger(lvl zapcore.Level) *ZapLogger {
	eConfig := zapcore.EncoderConfig{
		TimeKey:        "t",
		LevelKey:       "level",
		NameKey:        "logger",
		MessageKey:     "msg",
		StacktraceKey:  "stack",
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
	}

	return NewZapLogger(
		zapcore.NewConsoleEncoder(eConfig),
		zap.NewAtomicLevelAt(lvl),
		zap.AddStacktrace(zap.NewAtomicLevelAt(zapcore.ErrorLevel)),
	)
}

// InitJSONLogger creates a JSON logger.
func InitJSONLogger(lvl zapcore.Level) *ZapLogger {
	eConfig := zap.NewProductionEncoderConfig()
	eConfig.EncodeDuration = zapcore.SecondsDurationEncoder
	eConfig.EncodeTime = timeEncoder
	eConfig.CallerKey = "" // We handle caller ourselves

	return NewZapLogger(
		zapcore.NewJSONEncoder(eConfig),
		zap.NewAtomicLevelAt(lvl),
		zap.AddStacktrace(zap.NewAtomicLevelAt(zapcore.ErrorLevel)),
	)
}

func timeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

// skipPatterns contains path patterns to skip when finding the caller.
var skipPatterns = []string{
	"go-kratos/kratos",
	"pkg/log/zap.go",
}

// getCaller returns the caller information, skipping framework code.
// It returns file path relative to module root and line number.
func getCaller() string {
	const maxDepth = 15
	for i := 3; i < maxDepth; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		// Skip framework code
		skip := false
		for _, pattern := range skipPatterns {
			if strings.Contains(file, pattern) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		// Convert to relative path if possible
		return formatCaller(file, line)
	}
	return "unknown"
}

// formatCaller formats file:line, using relative path from common markers.
func formatCaller(file string, line int) string {
	// Try to find common path markers and make it relative
	markers := []string{"/internal/", "/pkg/", "/cmd/", "/test/"}
	for _, marker := range markers {
		if idx := strings.LastIndex(file, marker); idx != -1 {
			return fmt.Sprintf("%s:%d", file[idx+1:], line)
		}
	}
	// Fallback: use the last two path components
	parts := strings.Split(file, "/")
	if len(parts) >= 2 {
		return fmt.Sprintf("%s/%s:%d", parts[len(parts)-2], parts[len(parts)-1], line)
	}
	return fmt.Sprintf("%s:%d", file, line)
}
