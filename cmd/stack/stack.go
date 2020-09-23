/*
Copyright 2020 The Stack Authors.
*/

package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/kruglovmax/stack/pkg/app"
	"github.com/kruglovmax/stack/pkg/consts"
	"github.com/kruglovmax/stack/pkg/log"
	"github.com/kruglovmax/stack/pkg/stack"

	"github.com/spf13/pflag"
)

const (
	version = "v0.6.0"
	product = "stack"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// Flag domain.
	fs := pflag.NewFlagSet("default", pflag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "DESCRIPTION\n")
		fmt.Fprintf(os.Stderr, "  stack is more than template tool.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "FLAGS\n")
		fs.PrintDefaults()
	}

	versionFlag := fs.Bool("version", false, "get version number")

	app.App.Config.LogFormat = fs.StringP("log-format", "l", "fmt", "change the log format(json, fmt).")
	app.App.Config.Verbosity = fs.CountP("verb", "v", "verbosity")
	app.App.Config.CLIValues = fs.StringSliceP("set", "s", []string{}, `Additional vars
Example:
--set="name=value,topname.subname=value"`)
	app.App.Config.VarFiles = fs.StringSliceP("file", "f", []string{}, `Files with additional vars
Example:
-f vars.yaml -f vars2.yaml`)
	app.App.Config.Workdir = fs.StringP("workdir", "w", ".", `Working directory
Example:
--workdir="stackDir"
or
-w stackDir`)
	app.App.Config.GitLibsPath = fs.String("gitlibs-path", consts.GitLibsPath, `Directory where to clone libs from git
Example:
--gitlibs-path=".libs"`)
	app.App.Config.DefaultTimeout = fs.Duration("wait-timeout", consts.DefaultTimeout,
		"duration after which sync operations time out")

	err := fs.Parse(os.Args[1:])
	switch {
	case err == pflag.ErrHelp:
		os.Exit(0)
	case err != nil:
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", err.Error())
		fs.Usage()
		os.Exit(2)
	case *versionFlag:
		fmt.Println(version)
		os.Exit(0)
	}

	log.SetFormat(*app.App.Config.LogFormat)

	log.SetLevel(*app.App.Config.Verbosity)

	pwd, err := os.Getwd()
	if err != nil {
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg(err.Error())
	}
	if !filepath.IsAbs(*app.App.Config.Workdir) {
		pwd := filepath.Join(pwd, *app.App.Config.Workdir)
		app.App.Config.Workdir = &pwd
	}
	if err := os.Chdir(*app.App.Config.Workdir); err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(*app.App.Config.Workdir))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg(err.Error())
	}

	stack.RunRootStack(*app.App.Config.Workdir)

	os.Exit(0)
}
