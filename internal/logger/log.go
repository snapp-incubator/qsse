package logger

import (
	"log"
	"log/syslog"
	"os"

	"github.com/tchap/zapext/v2/zapsyslog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	Level   = "debug"
	Enabled = false
	Network = ""
	Address = ""
	Tag     = ""
)

func New() *zap.Logger {
	var lvl zapcore.Level
	if err := lvl.Set(Level); err != nil {
		log.Printf("cannot parse log level %s: %s", Level, err)

		lvl = zapcore.WarnLevel
	}

	encoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	defaultCore := zapcore.NewCore(encoder, zapcore.Lock(zapcore.AddSync(os.Stderr)), lvl)
	cores := []zapcore.Core{
		defaultCore,
	}

	if Enabled {
		p := getPriorityFromLevel(lvl.String()) | syslog.LOG_LOCAL0
		encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())

		writer, err := syslog.Dial(Network, Address, p, Tag)
		if err == nil {
			cores = append(cores, zapsyslog.NewCore(lvl, encoder, writer))
		} else {
			log.Printf("cannot create syslog core, error: %s", err.Error())
			log.Println("warning, logger output is only stdout")
		}
	}

	core := zapcore.NewTee(cores...)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))

	return logger
}

func getPriorityFromLevel(level string) syslog.Priority {
	switch level {
	case "debug":
		return syslog.LOG_DEBUG
	case "info":
		return syslog.LOG_INFO
	case "warn":
		return syslog.LOG_WARNING
	case "error":
		return syslog.LOG_ERR
	case "fatal":
		return syslog.LOG_CRIT
	case "panic":
		return syslog.LOG_ALERT

	default:
		return syslog.LOG_ERR
	}
}
