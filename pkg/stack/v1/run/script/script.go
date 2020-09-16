package script

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/Jeffail/gabs"
	"github.com/davecgh/go-spew/spew"
	"github.com/imdario/mergo"
	"github.com/joeycumines/go-dotnotation/dotnotation"
	"github.com/kruglovmax/stack/pkg/app"
	"github.com/kruglovmax/stack/pkg/conditions"
	"github.com/kruglovmax/stack/pkg/log"
	"github.com/kruglovmax/stack/pkg/misc"
	"github.com/kruglovmax/stack/pkg/types"
	"gopkg.in/yaml.v2"
)

// scriptItem type
type scriptItem struct {
	Script string        `json:"script,omitempty"`
	Vars   interface{}   `json:"vars,omitempty"`
	Output []interface{} `json:"output,omitempty"`
	When   string        `json:"when,omitempty"`
	Wait   string        `json:"wait,omitempty"`
}

// Exec func
func (item *scriptItem) Exec(parentWG *sync.WaitGroup, stack types.Stack, workdir string) {
	if parentWG != nil {
		defer parentWG.Done()
	}
	if !conditions.When(stack, item.When) {
		return
	}
	if !conditions.Wait(stack, item.Wait) {
		return
	}

	varsFile, err := ioutil.TempFile("/tmp", "vars")
	misc.CheckIfErr(err)
	defer os.Remove(varsFile.Name())
	switch item.Vars.(type) {
	case map[string]interface{}:
		err := ioutil.WriteFile(varsFile.Name(), []byte(misc.ToJSON(item.Vars.(map[string]interface{}))), 0600)
		misc.CheckIfErr(err)
	case string:
		vars, err := dotnotation.Get(stack.GetView(), item.Vars.(string))
		misc.CheckIfErr(err)
		err = ioutil.WriteFile(varsFile.Name(), []byte(misc.ToJSON(vars)), 0600)
		misc.CheckIfErr(err)
	case nil:
		err := ioutil.WriteFile(varsFile.Name(), []byte(misc.ToJSON(stack.GetView())), 0600)
		misc.CheckIfErr(err)
	default:
		log.Logger.Trace().
			Msg(spew.Sdump(item))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg("Unable to parse run item. Bad vars key")
	}
	cmd := exec.Command("sh", "-c", item.Script)
	cmd.Dir = stack.GetWorkdir()
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("VARS=%s", varsFile.Name()),
	)
	stderr, err := cmd.StderrPipe()
	misc.CheckIfErr(err)
	stdout, err := cmd.StdoutPipe()
	misc.CheckIfErr(err)

	stdoutBufio := bufio.NewScanner(stdout)
	stderrBufio := bufio.NewScanner(stderr)

	var wg sync.WaitGroup
	wg.Add(2)
	go item.getScriptOutput(stack, stdoutBufio, &wg, false)
	go item.getScriptOutput(stack, stderrBufio, &wg, true)

	err = cmd.Start()
	misc.CheckIfErr(err)

	if misc.WaitTimeout(&wg, *app.App.Config.WaitTimeout) {
		log.Logger.Fatal().
			Str("stack", stack.GetWorkdir()).
			Str("timeout", fmt.Sprint(*app.App.Config.WaitTimeout)).
			Msg("Script waiting failed")
	}

	err = cmd.Wait()
	misc.CheckIfErr(err)
}

func (item *scriptItem) getScriptOutput(stack types.Stack, output *bufio.Scanner, wg *sync.WaitGroup, isErr bool) {
	defer wg.Done()
	var outBuffer strings.Builder
	yml2var := ""
	str2var := ""
	outputType := item.Output
	stdoutMessagesChannel, stdoutListenerChannel := app.App.StdOut.StartOutputForObject()
	stderrMessagesChannel, stderrListenerChannel := app.App.StdErr.StartOutputForObject()

	for output.Scan() {
		line := output.Text()
		if isErr {
			log.Logger.Error().Msg("SCRIPT STDERR: " + line)
		} else if outputType != nil {
			for _, v := range outputType {
				switch v.(type) {
				case string:
					if v.(string) == "stdout" {
						app.App.StdOut.SendStringForObject(stdoutMessagesChannel, line)
					} else if v.(string) == "stderr" {
						app.App.StdErr.SendStringForObject(stderrMessagesChannel, line)
					}
				case map[string]interface{}:
					switch {
					case v.(map[string]interface{})["yml2var"] != nil:
						if outBuffer.Len() == 0 {
							outBuffer.WriteString(line)
						} else {
							outBuffer.WriteString("\n" + line)
						}
						yml2var = v.(map[string]interface{})["yml2var"].(string)
					case v.(map[string]interface{})["str2var"] != nil:
						if outBuffer.Len() == 0 {
							outBuffer.WriteString(line)
						} else {
							outBuffer.WriteString("\n" + line)
						}
						str2var = v.(map[string]interface{})["str2var"].(string)
					}
				}
			}
		}
	}

	if yml2var != "" {
		var value map[string]interface{}
		err := yaml.Unmarshal([]byte(outBuffer.String()), &value)
		misc.CheckIfErr(err)
		switch {
		case strings.HasPrefix(yml2var, "vars"):
			key := strings.TrimPrefix(strings.TrimPrefix(yml2var, "vars"), ".")
			setVar := gabs.New()
			if key == "" {
				setVar.Set(value)
			} else {
				setVar.SetP(value, key)
			}
			stack.AddRawVarsRight(setVar.Data().(map[string]interface{}))
		case strings.HasPrefix(yml2var, "flags"):
			key := strings.TrimPrefix(strings.TrimPrefix(yml2var, "flags"), ".")
			setVar := gabs.New()
			if key == "" {
				setVar.Set(value)
			} else {
				setVar.SetP(value, key)
			}
			stack.GetFlags().Mux.Lock()
			mergo.Merge(&stack.GetFlags().Vars, setVar.Data().(map[string]interface{}), mergo.WithOverwriteWithEmptyValue)
			stack.GetFlags().Mux.Unlock()
		case strings.HasPrefix(yml2var, "locals"):
			key := strings.TrimPrefix(strings.TrimPrefix(yml2var, "locals"), ".")
			setVar := gabs.New()
			if key == "" {
				setVar.Set(value)
			} else {
				setVar.SetP(value, key)
			}
			stack.GetLocals().Mux.Lock()
			mergo.Merge(&stack.GetLocals().Vars, setVar.Data().(map[string]interface{}), mergo.WithOverwriteWithEmptyValue)
			stack.GetLocals().Mux.Unlock()
		default:
			log.Logger.Fatal().
				Str("yml2var", yml2var).
				Str("in stack", stack.GetWorkdir()).
				Msg("Bad output var")
		}
	}
	if str2var != "" {
		value := outBuffer.String()
		switch {
		case strings.HasPrefix(str2var, "vars."):
			key := strings.TrimPrefix(str2var, "vars.")
			setVar := gabs.New()
			setVar.SetP(value, key)
			stack.AddRawVarsRight(setVar.Data().(map[string]interface{}))
		case strings.HasPrefix(str2var, "flags."):
			key := strings.TrimPrefix(str2var, "flags.")
			setVar := gabs.New()
			setVar.SetP(value, key)
			stack.GetFlags().Mux.Lock()
			mergo.Merge(&stack.GetFlags().Vars, setVar.Data().(map[string]interface{}), mergo.WithOverwriteWithEmptyValue)
			stack.GetFlags().Mux.Unlock()
		case strings.HasPrefix(str2var, "locals."):
			key := strings.TrimPrefix(str2var, "locals.")
			setVar := gabs.New()
			setVar.SetP(value, key)
			stack.GetLocals().Mux.Lock()
			mergo.Merge(&stack.GetLocals().Vars, setVar.Data().(map[string]interface{}), mergo.WithOverwriteWithEmptyValue)
			stack.GetLocals().Mux.Unlock()
		default:
			log.Logger.Fatal().
				Str("str2var", str2var).
				Str("in stack", stack.GetWorkdir()).
				Msg("Bad output var")
		}
	}

	app.App.StdOut.FinishOutputForObject(stdoutMessagesChannel, stdoutListenerChannel)
	app.App.StdErr.FinishOutputForObject(stderrMessagesChannel, stderrListenerChannel)
}

// Parse func
func Parse(workDir string, item interface{}) types.RunItem {
	tmplItem := item.(map[string]interface{})
	output := new(scriptItem)
	output.Script = (item).(map[string]interface{})["script"].(string)
	output.Vars = tmplItem["vars"]
	output.Output = (item).(map[string]interface{})["output"].([]interface{})
	whenCondition := (item).(map[string]interface{})["when"]
	waitCondition := (item).(map[string]interface{})["wait"]
	if whenCondition != nil {
		output.When = whenCondition.(string)
	}
	if waitCondition != nil {
		output.Wait = waitCondition.(string)
	}

	return output
}
