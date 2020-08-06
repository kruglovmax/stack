package v1alpha1

import (
	"bufio"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/Jeffail/gabs"
	"github.com/davecgh/go-spew/spew"
	"github.com/kruglovmax/stack/pkg/log"
	"sigs.k8s.io/yaml"
)

// Run struct
type Run []runItem

type runItem interface {
	execute(stack *Stack)
	getOutput() []interface{}
}

// GetRunItemOutput func
func GetRunItemOutput(stack *Stack, item runItem, output *bufio.Scanner, done chan<- struct{}, isErr bool) {
	var outBuffer strings.Builder
	outVarYml := ""
	outVarStr := ""
	outputType := item.getOutput()

	for output.Scan() {
		line := output.Text()
		if isErr {
			log.Logger.Error().Msg("SCRIPT STDERR: " + line)
		} else if outputType != nil {
			for _, v := range outputType {
				switch v.(type) {
				case string:
					if v.(string) == "stdout" {
						fmt.Fprintln(os.Stdout, line)
					} else if v.(string) == "stderr" {
						fmt.Fprintln(os.Stderr, line)
					}
				case map[string]interface{}:
					switch {
					case v.(map[string]interface{})["ymlvar"] != nil:
						if outBuffer.Len() == 0 {
							outBuffer.WriteString(line)
						} else {
							outBuffer.WriteString("\n" + line)
						}
						outVarYml = v.(map[string]interface{})["ymlvar"].(string)
					case v.(map[string]interface{})["strvar"] != nil:
						if outBuffer.Len() == 0 {
							outBuffer.WriteString(line)
						} else {
							outBuffer.WriteString("\n" + line)
						}
						outVarStr = v.(map[string]interface{})["strvar"].(string)
					}
				}
			}
		}
	}

	if outVarYml != "" {
		var setValue map[string]interface{}
		key := strings.TrimRight("vars."+strings.TrimLeft(outVarYml, "."), ".")
		err := yaml.Unmarshal([]byte(outBuffer.String()), &setValue)
		if err != nil {
			log.Logger.Trace().
				Msg(spew.Sdump(outBuffer.String()))
			log.Logger.Debug().
				Msg(string(debug.Stack()))
			log.Logger.Fatal().
				Msg(err.Error())
		}

		setVar := gabs.New()
		setVar.SetP(setValue, key)
		stack.AddVarsRight(setVar.Data().(map[string]interface{})["vars"])

		log.Logger.Trace().
			Msg(spew.Sdump(stack.Vars))
		log.Logger.Debug().
			Msgf("Variable %s is setted to:\n%s", key, spew.Sdump(setVar.Data()))
	}
	if outVarStr != "" {
		key := strings.TrimRight("vars."+strings.TrimLeft(outVarStr, "."), ".")
		setValue := outBuffer.String()

		setVar := gabs.New()
		setVar.SetP(setValue, key)
		stack.AddVarsRight(setVar.Data().(map[string]interface{})["vars"])

		log.Logger.Debug().
			Msgf("Variable %s is setted to:\n%s", key, setValue)
		log.Logger.Trace().
			Msg(spew.Sdump(stack.Vars))
		log.Logger.Trace().
			Msg(spew.Sdump(setVar.Data()))
	}

	done <- struct{}{}
}
