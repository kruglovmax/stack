package types

import (
	"sync"
)

// Config interface
type Config interface {
	ToMap() map[string]interface{}
}

// Stack interface
type Stack interface {
	AddRawVarsLeft(map[string]interface{})
	AddRawVarsRight(map[string]interface{})
	Start(*sync.WaitGroup)
	PreExec(*sync.WaitGroup)
	Exec(*sync.WaitGroup)
	PostExec(*sync.WaitGroup)
	GetAPI() string
	GetLibs() []string
	GetName() string
	GetInput() *StackInput
	GetVars() *StackVars
	GetFlags() *StackFlags
	GetLocals() *StackLocals
	GetRunItemsParser() RunItemParser
	GetStackID() string
	GetView() interface{}
	GetWorkdir() string
	SetStatus(string)
	LoadFromFile(string, Stack)
	LoadFromString(string, Stack)
}

// StackVars type
type StackVars struct {
	Vars      map[string]interface{}
	Modifiers map[string]interface{}
	Mux       sync.Mutex
}

// StackFlags (global vars)
type StackFlags struct {
	Vars map[string]interface{}
	Mux  sync.Mutex
}

// StackLocals type
type StackLocals struct {
	Vars map[string]interface{}
	Mux  sync.Mutex
}

// StackInput type
type StackInput struct {
	Input interface{}
	Mux   sync.Mutex
}

// StackExitCode of stack
type StackExitCode struct {
	Status uint64
	Stack  Stack
}

// StacksStatus type
type StacksStatus struct {
	StacksStatus map[string]string
	Mux          sync.Mutex
}

// ExecExitCode of stack
type ExecExitCode struct {
	Status  uint64
	RunItem RunItem
}

// RunItem interface
type RunItem interface {
	Exec(*sync.WaitGroup, Stack)
}

// RunItemParser interface
type RunItemParser interface {
	ParseRun(Stack, []interface{}) (output []RunItem)
	ParseRunItem(Stack, interface{}) (output RunItem)
}
