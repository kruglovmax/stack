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
	"github.com/kruglovmax/stack/pkg/appconfig"
	"github.com/kruglovmax/stack/pkg/log"
	"github.com/kruglovmax/stack/pkg/misc"
	stack "github.com/kruglovmax/stack/pkg/stack"

	"github.com/spf13/pflag"
	"helm.sh/helm/v3/pkg/strvals"
)

const (
	version = "v1.0.0-alpha1"
	product = "stack"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	rootStack := stack.RootStack
	appConfig := new(appconfig.AppConfig)

	// Flag domain.
	fs := pflag.NewFlagSet("default", pflag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "DESCRIPTION\n")
		fmt.Fprintf(os.Stderr, "  stack is more than template tool.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "FLAGS\n")
		fs.PrintDefaults()
	}

	var (
		versionFlag = fs.Bool("version", false, "get version number")
	)

	appConfig.LogFormat = fs.StringP("log-format", "l", "fmt", "change the log format.")
	appConfig.Verbosity = fs.CountP("verb", "v", "verbosity")
	appConfig.TagPatterns = fs.StringSliceP("tags-patterns", "t", []string{}, `Stack tags
		Example:
		--tags-patterns="cluster,dev,eu-central-1"`)
	appConfig.CLIValues = fs.StringSliceP("set", "s", []string{}, `Additional vars
		Example:
		--set="name=value,topname.subname=value"`)
	appConfig.VarFiles = fs.StringSliceP("file", "f", []string{}, `Files with additional vars
		Example:
		-f vars.yaml -f vars2.yaml`)
	appConfig.Workspace = fs.StringP("workspace", "w", ".", `Working directory
		Example:
		--workspace="stackDir"
		Default: "."`)

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

	log.SetLevel(*appConfig.Verbosity)

	pwd, err := os.Getwd()
	if err != nil {
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg(err.Error())
	}
	if !filepath.IsAbs(*appConfig.Workspace) {
		pwd := filepath.Join(pwd, *appConfig.Workspace)
		appConfig.Workspace = &pwd
	}
	if err := os.Chdir(*appConfig.Workspace); err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(*appConfig.Workspace))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg(err.Error())
	}

	rootStack.FromFile(appConfig, "stack.yaml", nil)

	for _, file := range *appConfig.VarFiles {
		var vars interface{}
		misc.LoadYAMLFromSopsFile(file, &vars)
		rootStack.AddVarsLeft(vars)
	}

	for _, str := range *appConfig.CLIValues {
		vars, err := strvals.Parse(str)
		if err != nil {
			log.Logger.Trace().
				Msg(spew.Sdump(str))
			log.Logger.Debug().
				Msg(string(debug.Stack()))
			log.Logger.Fatal().
				Msg(err.Error())
		}
		rootStack.AddVarsLeft(vars)
	}

	rootStack.Execute()

	os.Exit(0)
}
