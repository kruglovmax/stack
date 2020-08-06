package v1alpha1

import "github.com/kruglovmax/stack/pkg/misc"

/*
CombineStack func
*/
func (stack *Stack) combineStack() (err error) {
	if stack.parentStack != nil {
		stack.appConfig = stack.parentStack.appConfig
		stack.Vars = misc.CombineVars(stack.parentStack.Vars, stack.Vars)
		stack.Tags = misc.UniqueStrings(append(stack.Tags, stack.parentStack.Tags...))
	}

	return
}
