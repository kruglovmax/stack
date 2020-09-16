package log

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Type type
type Type uint32

// Type consts
const (
	PrettyType Type = 1
	JSONType
)

func (e Type) String() string {
	return [...]string{
		"",
		"pretty",
		"json",
	}[e]
}

var levels = map[interface{}]zerolog.Level{
	"panic": zerolog.PanicLevel,
	"fatal": zerolog.FatalLevel,
	"error": zerolog.ErrorLevel,
	"warn":  zerolog.WarnLevel,
	"info":  zerolog.InfoLevel,
	"debug": zerolog.DebugLevel,
	"trace": zerolog.TraceLevel,
	0:       zerolog.FatalLevel,
	1:       zerolog.InfoLevel,
	2:       zerolog.DebugLevel,
	3:       zerolog.TraceLevel,
}

// Logger var
var Logger zerolog.Logger

// SetLevel func
// "panic": zerolog.PanicLevel
// "fatal": zerolog.FatalLevel
// "error": zerolog.ErrorLevel
// "warn":  zerolog.WarnLevel
// "info":  zerolog.InfoLevel
// "debug": zerolog.DebugLevel
// "trace": zerolog.TraceLevel
func SetLevel(level interface{}) {
	if _, ok := levels[level]; ok {
		zerolog.SetGlobalLevel(levels[level])
	} else {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	}
}

// SetFormat func
func SetFormat(format string) {
	switch format {
	case "json":
		Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	case "fmt":
		Logger = zerolog.New(
			zerolog.ConsoleWriter{
				Out:        os.Stderr,
				TimeFormat: time.RFC3339Nano,
			}).With().Timestamp().Logger()
	default:
		Logger = zerolog.New(
			zerolog.ConsoleWriter{
				Out:        os.Stderr,
				TimeFormat: time.RFC3339Nano,
			}).With().Timestamp().Logger()
	}
}

func init() {
	Logger = zerolog.New(
		zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339Nano,
		}).With().Timestamp().Logger()
}
