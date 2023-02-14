package logger

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger struct {
	*zap.Logger
}

var (
	loggers = make(map[string]*Logger)
	mu      sync.RWMutex
)

var Runtime = Get("runtime")

func Get(filename string) *Logger {
	mu.RLock()
	if l, ok := loggers[filename]; ok {
		mu.RUnlock()
		return l
	}
	mu.RUnlock()

	// 实例锁
	mu.Lock()
	defer mu.Unlock()
	if l, ok := loggers[filename]; ok {
		return l
	}
	encoder := getEncoder()
	writeSyncer := getLogWriter(fmt.Sprintf("%s/%s/", "log", filename))
	core := zapcore.NewCore(encoder, writeSyncer, zapcore.DebugLevel)
	loggers[filename] = &Logger{zap.New(core, zap.AddCaller())}
	return loggers[filename]
}

func getLogWriter(path string) zapcore.WriteSyncer {
	fileName := path + "log"
	lumberJackLogger := &lumberjack.Logger{
		Filename:   fileName,
		MaxSize:    10, // 10M切割
		MaxAge:     5,  // 保留旧文件个数
		MaxBackups: 30, // 旧文件存活天数
		Compress:   true,
	}
	return zapcore.AddSync(lumberJackLogger)
}
func getEncoder() zapcore.Encoder {
	//return zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = logTimeFormat
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}
func logTimeFormat(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("[2006-01-02 15:04:05]"))
}

func (l *Logger) Debug(msg string) {
	l.Logger.Debug(msg, l.locateField())
}

func (l *Logger) Info(msg string) {
	l.Logger.Info(msg)
}
func (l *Logger) Warn(msg string) {
	l.Logger.Warn(msg)
}
func (l *Logger) Error(msg string) {
	l.Logger.Error(msg, l.locateField())
}

func (l *Logger) locateField() zap.Field {
	fileLine := ""
	fileLineSlice := make([]string, 0)
	for i := 0; i < 10; i++ {
		if _, file, line, ok := runtime.Caller(i); ok && file != "" {
			if binPath, err := os.Getwd(); err == nil {
				file = strings.ReplaceAll(file, binPath, "")
			}
			fileLineSlice = append(fileLineSlice, fmt.Sprintf("%s:%d", file, line))
		}
	}
	if len(fileLineSlice) > 0 {
		fileLine = strings.Join(fileLineSlice, "  |  ")
	}
	return zap.String("_caller", fileLine)
}

//for io.writer
func (l *Logger) Write(msg []byte) (int, error) {
	l.Logger.Debug(string(msg))
	return len(msg), nil
}
