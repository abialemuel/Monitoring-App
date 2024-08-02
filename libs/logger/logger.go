package logger

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

type Config struct {
	Level  string
	Format string
}

type Logger interface {
	Init(conf Config) Logger
	Get() *logrus.Logger
	UseForSystemLog() Logger
}

type logger struct {
	log *logrus.Logger
}

func New() Logger {
	return &logger{log: logrus.New()}
}

func (c *logger) UseForSystemLog() Logger {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	log.SetOutput(c.log.Writer())
	return c
}

func (c *logger) Init(conf Config) Logger {
	c.log.SetReportCaller(true)
	switch conf.Format {
	case "json":
		c.json()
	default:
		c.text()
	}
	if level, err := logrus.ParseLevel(conf.Level); err == nil {
		c.log.SetLevel(level)
	}
	return c
}

func (c *logger) text() {
	c.log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:    true,
		TimestampFormat:  "15:04:05.000",
		CallerPrettyfier: c.callerPretty("text"),
	})
}

func (c *logger) json() {
	c.log.SetFormatter(&logrus.JSONFormatter{
		CallerPrettyfier: c.callerPretty("json"),
	})
}

func (c *logger) Get() *logrus.Logger {
	return c.log
}

func (c *logger) callerPretty(format string) func(frame *runtime.Frame) (function string, file string) {
	return func(frame *runtime.Frame) (function string, file string) {
		fileShort := frame.File
		_, b, _, _ := runtime.Caller(0)
		rel, err := filepath.Rel(filepath.Dir(b), frame.File)
		if err == nil {
			switch runtime.GOOS {
			case "windows":
			default:
				fileShort = ":" + strings.ReplaceAll(rel, "../", "")
			}
		}
		if format == "json" {
			return function, fmt.Sprintf("%s:%d", fileShort, frame.Line)
		}
		return function, fmt.Sprintf("[%s:%d]", fileShort, frame.Line)
	}
}
