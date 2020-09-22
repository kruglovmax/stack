package app

import (
	"os"
	"sync"
	"time"

	"github.com/kruglovmax/stack/pkg/out"
)

// App global var
var (
	App *app
)

// consts
const (
	GitLibsPath = ".gitlibs"
)

type app struct {
	Config *appConfig
	Mutex  *appMutex
	StdOut *out.Output
	StdErr *out.Output
}

type appConfig struct {
	CLIValues      *[]string
	LogFormat      *string        `json:"LogFormat,omitempty"`
	VarFiles       *[]string      `json:"VarFiles,omitempty"`
	Verbosity      *int           `json:"Verbosity,omitempty"`
	DefaultTimeout *time.Duration `json:"DefaultTimeout,omitempty"`
	Workdir        *string        `json:"Workdir,omitempty"`
}

type appMutex struct {
	CurrentWorkDirMutex sync.Mutex
	GitWorkMutex        sync.Mutex
}

func init() {
	App = new(app)
	App.Config = new(appConfig)
	App.Mutex = new(appMutex)
	App.StdOut = out.New(os.Stdout)
	App.StdErr = out.New(os.Stderr)
}
