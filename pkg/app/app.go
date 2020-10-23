package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kruglovmax/stack/pkg/consts"
	"github.com/kruglovmax/stack/pkg/log"
	"github.com/kruglovmax/stack/pkg/out"
	"github.com/kruglovmax/stack/pkg/types"
)

// App global var
var (
	App *app
)

type app struct {
	Context       context.Context
	Cancel        context.CancelFunc
	Config        *appConfig
	StacksStatus  *types.StacksStatus
	Mutex         *appMutex
	StacksCounter uint64
	StdOut        *out.Output
	StdErr        *out.Output
	WaitGroups    map[string]*sync.WaitGroup
	AppError      int
}

type appConfig struct {
	CLIValues      *[]string
	LogFormat      *string        `json:"LogFormat,omitempty"`
	VarFiles       *[]string      `json:"VarFiles,omitempty"`
	Verbosity      *int           `json:"Verbosity,omitempty"`
	DefaultTimeout *time.Duration `json:"DefaultTimeout,omitempty"`
	Workdir        *string        `json:"Workdir,omitempty"`
	GitLibsPath    *string        `json:"GitLibsPath,omitempty"`
}

type appMutex struct {
	CurrentWorkDirMutex sync.Mutex
	GitWorkMutex        sync.Mutex
	StacksCounterMutex  sync.Mutex
}

func init() {
	App = new(app)
	App.Context, App.Cancel = context.WithCancel(context.Background())
	App.Config = new(appConfig)
	App.Mutex = new(appMutex)
	App.StdOut = out.New(os.Stdout)
	App.StdErr = out.New(os.Stderr)
	App.StacksStatus = new(types.StacksStatus)
	App.StacksStatus.StacksStatus = make(map[string]string)
	App.StacksCounter = 0
	App.WaitGroups = make(map[string]*sync.WaitGroup)

	setupCloseHandler()
}

// NewStackID func
func NewStackID() string {
	App.Mutex.StacksCounterMutex.Lock()
	App.StacksCounter++
	App.Mutex.StacksCounterMutex.Unlock()
	return fmt.Sprintf("stack_%v", App.StacksCounter)
}

func setupCloseHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Logger.Error().Msg("SIGTERM received. Gracefully shutting down...")
		App.AppError = consts.ExitCodeSIGTERM
		App.Cancel()
	}()
}
