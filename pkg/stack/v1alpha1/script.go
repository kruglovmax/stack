package v1alpha1

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime/debug"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/kruglovmax/stack/pkg/log"
	"github.com/kruglovmax/stack/pkg/misc"
)

type scriptItem struct {
	item  *interface{}
	stack *Stack

	Script  string        `json:"script,omitempty"`
	Vars    interface{}   `json:"vars,omitempty"`
	Output  []interface{} `json:"output,omitempty"`
	Timeout uint64        `json:"timeout,omitempty"`
}

func (item scriptItem) getOutput() []interface{} {
	return item.Output
}

// Execute func
func (item scriptItem) execute(stack *Stack) {
	var err error
	var varsFile *os.File

	varsFile, err = ioutil.TempFile("/tmp", "vars")
	if err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(item))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg("Temporary file create failure in dir /tmp\n" + err.Error())
	}
	defer os.Remove(varsFile.Name())

	err = ioutil.WriteFile(varsFile.Name(), []byte(misc.ToJSON(item.RootMap(*stack))), 0600)
	if err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(item))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg("Temporary file write failure\n" + err.Error())
	}

	cmd := exec.Command("sh", "-c", item.Script)
	cmd.Dir = stack.Workspace
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("VARS=%s", varsFile.Name()),
	)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(item))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg("Can't attach to stderr pipe\n" + err.Error())
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(item))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg("Can't attach to stdout pipe\n" + err.Error())
	}

	stdoutReadDone := make(chan struct{})
	stderrReadDone := make(chan struct{})

	stdoutBufio := bufio.NewScanner(stdout)
	stderrBufio := bufio.NewScanner(stderr)

	go GetRunItemOutput(stack, item, stdoutBufio, stdoutReadDone, false)
	go GetRunItemOutput(stack, item, stderrBufio, stderrReadDone, true)

	if err := cmd.Start(); err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(item))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg("Script run failure\n" + err.Error())
	}

	switch {
	case item.Timeout > 0:
		select {
		case <-stdoutReadDone:
		case <-time.After(time.Duration(item.Timeout) * time.Second):
			log.Logger.Trace().
				Msg(spew.Sdump(item))
			log.Logger.Debug().
				Msg(string(debug.Stack()))
			log.Logger.Fatal().
				Msg("Script run timeout")
		}
	default:
		<-stdoutReadDone
	}
	switch {
	case item.Timeout > 0:
		select {
		case <-stderrReadDone:
		case <-time.After(time.Duration(item.Timeout) * time.Second):
			log.Logger.Trace().
				Msg(spew.Sdump(item))
			log.Logger.Debug().
				Msg(string(debug.Stack()))
			log.Logger.Fatal().
				Msg("Script run timeout")
		}
	default:
		<-stderrReadDone
	}

	if err := cmd.Wait(); err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(item))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg("Script run failure\n" + err.Error())
	}

	log.Logger.Trace().
		Msg(spew.Sdump(stack.GetRealVars()))

}

func (item scriptItem) RootMap(stack Stack) interface{} {
	vars := GetRunItemVars(stack, item.Vars)
	if vars != nil {
		return misc.GetRealVars(vars)
	}
	return nil
}

func parseScriptItem(stack *Stack, item *interface{}) (result scriptItem) {
	// vars := GetRunItemVars(stack, item)
	outputType := misc.GetRunItemOutputType(*item)
	sItem := (*item).(map[string]interface{})
	var timeout uint64

	if sItem["timeout"] != nil {
		timeout = uint64(sItem["timeout"].(float64))
	}

	if err := os.Chdir(stack.Workspace); err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(stack.Workspace))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg(err.Error())
	}
	result = scriptItem{
		stack:   stack,
		item:    item,
		Script:  processString(stack.ToMap(), sItem["script"].(string)),
		Timeout: timeout,
		Vars:    sItem["vars"],
		Output:  outputType,
	}
	os.Chdir(*stack.appConfig.Workspace)
	return
}
