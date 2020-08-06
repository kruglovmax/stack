package v1alpha1

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/davecgh/go-spew/spew"
	"github.com/kruglovmax/stack/pkg/log"
	"github.com/kruglovmax/stack/pkg/misc"
)

const (
	chartTimeout = 60 //seconds
)

type chartItem struct {
	stack              *Stack
	scriptItemInstance scriptItem

	Chart     string        `json:"chart,omitempty"`
	Namespace string        `json:"namespace,omitempty"`
	Name      string        `json:"name,omitempty"`
	Repo      string        `json:"repo,omitempty"`
	Vars      interface{}   `json:"vars,omitempty"`
	Output    []interface{} `json:"output,omitempty"`
	Timeout   uint64        `json:"timeout,omitempty"`
}

// Execute func
func (item chartItem) execute(stack *Stack) {
	item.scriptItemInstance.execute(stack)
}

func (item chartItem) getOutput() []interface{} {
	return item.scriptItemInstance.Output
}

func parseChartItem(stack *Stack, item *interface{}) chartItem {
	var scriptHelmTemplate string

	outputType := misc.GetRunItemOutputType(*item)
	sItem := (*item).(map[string]interface{})

	if err := os.Chdir(stack.Workspace); err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(stack.Workspace))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg(err.Error())
	}
	switch {
	case sItem["repo"] != nil:
		scriptHelmTemplate = fmt.Sprintf("jq -r . $VARS | helm template %s %s "+
			"--namespace %s --repo '%s'",
			processString(stack.ToMap(), sItem["name"].(string)),
			processString(stack.ToMap(), sItem["chart"].(string)),
			processString(stack.ToMap(), sItem["namespace"].(string)),
			processString(stack.ToMap(), sItem["repo"].(string)),
		)
		if sItem["version"] != nil {
			scriptHelmTemplate = scriptHelmTemplate +
				" --version " +
				processString(stack.ToMap(), sItem["version"].(string))
		}

		scriptHelmTemplate = scriptHelmTemplate +
			" > stack-output-for-kustomize.yaml; " +
			fmt.Sprintf("echo '"+
				"apiVersion: kustomize.config.k8s.io/v1beta1\n"+
				"kind: Kustomization\n"+
				"namespace: %s\n"+
				"resources:\n"+
				"- stack-output-for-kustomize.yaml' > kustomization.yaml",
				processString(stack.ToMap(), sItem["namespace"].(string))) +
			"; kubectl kustomize ." +
			"; rm stack-output-for-kustomize.yaml" +
			"; rm kustomization.yaml"
	case sItem["repo"] == nil:
		scriptHelmTemplate = fmt.Sprintf("jq -r . $VARS | helm template %s %s "+
			"--namespace %s",
			processString(stack.ToMap(), sItem["name"].(string)),
			processStringPath(*stack, sItem["chart"].(string)),
			processString(stack.ToMap(), sItem["namespace"].(string)),
		)

		scriptHelmTemplate = scriptHelmTemplate +
			" > stack-output-for-kustomize.yaml; " +
			fmt.Sprintf("echo '"+
				"apiVersion: kustomize.config.k8s.io/v1beta1\n"+
				"kind: Kustomization\n"+
				"namespace: %s\n"+
				"resources:\n"+
				"- stack-output-for-kustomize.yaml' > kustomization.yaml",
				processString(stack.ToMap(), sItem["namespace"].(string))) +
			"; kubectl kustomize ." +
			"; rm stack-output-for-kustomize.yaml" +
			"; rm kustomization.yaml"
	}
	os.Chdir(*stack.appConfig.Workspace)

	out := scriptItem{
		stack:   stack,
		item:    item,
		Script:  scriptHelmTemplate,
		Output:  outputType,
		Vars:    sItem["vars"],
		Timeout: chartTimeout,
	}
	repo := ""
	if sItem["repo"] != nil {
		repo = sItem["repo"].(string)
	}
	return chartItem{
		stack:              stack,
		scriptItemInstance: out,
		Chart:              sItem["chart"].(string),
		Namespace:          sItem["namespace"].(string),
		Name:               sItem["name"].(string),
		Vars:               sItem["vars"],
		Repo:               repo,
		Output:             outputType,
	}
}
