package stack

import (
	"io/ioutil"
	"runtime/debug"

	"github.com/davecgh/go-spew/spew"
	"github.com/kruglovmax/stack/pkg/consts"
	"github.com/kruglovmax/stack/pkg/log"
	"github.com/kruglovmax/stack/pkg/misc"
	v1 "github.com/kruglovmax/stack/pkg/stack/v1/stack"
	"github.com/kruglovmax/stack/pkg/types"
)

// RootStack instance
var rootStack types.Stack

// GetRootStack func
func GetRootStack() types.Stack {
	return rootStack
}

// RunRootStack func
func RunRootStack(workdir string) {
	var preConfig interface{}

	stackFile := misc.FindStackFileInDir(workdir)

	content, ioErr := ioutil.ReadFile(stackFile)
	if ioErr != nil {
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Str("file", stackFile).
			Msg(ioErr.Error())
	}

	misc.LoadYAML(string(content), &preConfig)

	switch preConfig.(type) {
	case map[string]interface{}:
		switch preConfig.(map[string]interface{})["api"] {
		case "v1":
			rootStack = new(v1.Stack)
			rootStack.LoadFromFile(stackFile, nil)
			rootStack.Start(nil)
		default:
			log.Logger.Debug().
				Msg(string(debug.Stack()))
			log.Logger.Fatal().
				Str("file", stackFile).
				Str("api", spew.Sdump(preConfig.(map[string]interface{})["api"])).
				Msg(consts.MessageBadStackUnsupportedAPI)
		}
	default:
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Str("file", stackFile).
			Msg(consts.MessageBadStack)
	}
}
