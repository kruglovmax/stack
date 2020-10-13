package vars

import (
	"runtime/debug"

	"github.com/davecgh/go-spew/spew"
	"github.com/kruglovmax/stack/pkg/consts"
	"github.com/kruglovmax/stack/pkg/log"
	"github.com/kruglovmax/stack/pkg/types"
)

// StackVarsModifiers type
type StackVarsModifiers struct {
	Update bool
	Clear  bool
	Weak   bool
}

var (
	// FlagsGlobal var
	FlagsGlobal *types.StackFlags

	// VarsModifiersSuffix var
	VarsModifiersSuffix = "_modifiers"

	// VarsSuffixDelimeter var
	VarsSuffixDelimeter = "^"

	// VarsSuffixes var
	VarsSuffixes = map[string]string{
		"Update": "+",
		"Clear":  "-",
		"Weak":   "~",
	}
)

func init() {
	if FlagsGlobal == nil {
		FlagsGlobal = new(types.StackFlags)
		if FlagsGlobal.Vars == nil {
			FlagsGlobal.Vars = make(map[string]interface{})
		}
	}
}

// ParseVars func
func ParseVars(varsFromConfig map[string]interface{}) *types.StackVars {
	return parseVars(varsFromConfig, nil)
}

func parseVars(varsFromConfig map[string]interface{}, parentVarModifiers *StackVarsModifiers) (stackVars *types.StackVars) {
	stackVars = new(types.StackVars)
	modifiers := make(map[string]interface{})
	vars := make(map[string]interface{})

	for varRawName, varValue := range varsFromConfig {
		varName, varModifiers := ParseVarModifiers(varRawName, parentVarModifiers)
		switch varValue.(type) {
		case map[string]interface{}:
			stackSubVars := parseVars(varValue.(map[string]interface{}), &varModifiers)
			_, ok := modifiers[varName]
			if ok {
				log.Logger.Trace().
					Msg(spew.Sdump(vars))
				log.Logger.Debug().
					Msg(string(debug.Stack()))
				log.Logger.Fatal().
					Str("var", varName).
					Msg(consts.MessageVarsDoubleDefinition)
			}
			modifiers[varName] = stackSubVars.Modifiers
			modifiers[varName+VarsModifiersSuffix] = varModifiers
			vars[varName] = stackSubVars.Vars
		default:
			_, ok := modifiers[varName]
			if ok {
				log.Logger.Trace().
					Msg(spew.Sdump(vars))
				log.Logger.Debug().
					Msg(string(debug.Stack()))
				log.Logger.Fatal().
					Str("var", varName).
					Msg(consts.MessageVarsDoubleDefinition)
			}
			modifiers[varName+VarsModifiersSuffix] = varModifiers
			vars[varName] = varValue
		}
	}
	stackVars.Vars = vars
	stackVars.Modifiers = modifiers
	return
}

// ParseVarModifiers func
// varRawName - test+
func ParseVarModifiers(varRawName string, parentVarModifiers *StackVarsModifiers) (varName string, modifiers StackVarsModifiers) {
	varName = varRawName
	if parentVarModifiers != nil {
		modifiers.Weak = parentVarModifiers.Weak
	}
Loop:
	for {
		sfx := varName[len(varName)-1:]
		varName = varName[:len(varName)-1]
		switch sfx {
		case VarsSuffixes["Update"]:
			if !(modifiers.Update ||
				modifiers.Weak) &&
				(parentVarModifiers == nil || (parentVarModifiers != nil && parentVarModifiers.Update)) {
				if modifiers.Clear {
					log.Logger.Debug().
						Msg(string(debug.Stack()))
					log.Logger.Warn().
						Str("Input var name", varRawName).
						Str("Output var name", varName).
						Msgf(consts.MessageVarsSimplyfy, varName)
				}
				modifiers.Update = true
			} else {
				varName = varName + sfx
				log.Logger.Debug().
					Msg(string(debug.Stack()))
				log.Logger.Warn().
					Str("Input var name", varRawName).
					Str("Output var name", varName).
					Msg(consts.MessageVarsBadVarName)
				break Loop
			}
		case VarsSuffixes["Clear"]:
			if !modifiers.Clear &&
				parentVarModifiers == nil {
				modifiers.Clear = true
				if modifiers.Update {
					log.Logger.Debug().
						Msg(string(debug.Stack()))
					log.Logger.Warn().
						Str("Input var name", varRawName).
						Str("Output var name", varName).
						Msgf(consts.MessageVarsSimplyfy, varName)
				}
			} else {
				varName = varName + sfx
				log.Logger.Debug().
					Msg(string(debug.Stack()))
				log.Logger.Warn().
					Str("Input var name", varRawName).
					Str("Output var name", varName).
					Msg(consts.MessageVarsBadVarName)
				break Loop
			}
		case VarsSuffixes["Weak"]:
			if !(modifiers.Update ||
				modifiers.Weak) &&
				parentVarModifiers == nil {
				modifiers.Weak = true
			} else {
				varName = varName + sfx
				log.Logger.Debug().
					Msg(string(debug.Stack()))
				log.Logger.Warn().
					Str("Input var name", varRawName).
					Str("Output var name", varName).
					Msg(consts.MessageVarsBadVarName)
				break Loop
			}
		case VarsSuffixDelimeter:
			break Loop
		default:
			varName = varName + sfx
			break Loop
		}
	}
	return
}

// CombineVars func
func CombineVars(leftVars, rightVars *types.StackVars) (combinedVars *types.StackVars) {
	combinedVars = new(types.StackVars)
	combinedVars.Vars = rightVars.Vars
	combinedVars.Modifiers = rightVars.Modifiers
	if combinedVars.Vars == nil {
		combinedVars.Vars = make(map[string]interface{})
	}
	if combinedVars.Modifiers == nil {
		combinedVars.Modifiers = make(map[string]interface{})
	}

	for leftKey, leftValue := range leftVars.Vars {
		rightValue := rightVars.Vars[leftKey]
		leftVarsModifiers := leftVars.Modifiers[leftKey+VarsModifiersSuffix].(StackVarsModifiers)
		rightVarsModifiers, ok := rightVars.Modifiers[leftKey+VarsModifiersSuffix].(StackVarsModifiers)
		if !ok {
			rightVarsModifiers = StackVarsModifiers{Weak: true}
		}
		switch {
		case leftVarsModifiers.Weak:
			if rightVarsModifiers.Clear {
				combinedVars.Vars[leftKey] = rightValue
				combinedVars.Modifiers[leftKey] = nil
				combinedVars.Modifiers[leftKey+VarsModifiersSuffix] = rightVarsModifiers
			} else {
				switch rightValue.(type) {
				case []interface{}:
					switch leftValue.(type) {
					case []interface{}:
						combinedVars.Vars[leftKey] = append(leftValue.([]interface{}), rightValue.([]interface{})...)
						combinedVars.Modifiers[leftKey+VarsModifiersSuffix] = rightVarsModifiers
						combinedVars.Modifiers[leftKey] = rightVars.Modifiers[leftKey]
					default:
						combinedVars.Vars[leftKey] = rightValue
						combinedVars.Modifiers[leftKey+VarsModifiersSuffix] = rightVarsModifiers
						combinedVars.Modifiers[leftKey] = rightVars.Modifiers[leftKey]
					}
				case map[string]interface{}:
					switch leftValue.(type) {
					case map[string]interface{}:
						newLeftVars := new(types.StackVars)
						switch leftVars.Modifiers[leftKey].(type) {
						case map[string]interface{}:
							newLeftVars.Modifiers = leftVars.Modifiers[leftKey].(map[string]interface{})
						}
						newLeftVars.Vars = leftValue.(map[string]interface{})
						newRightVars := new(types.StackVars)
						switch rightVars.Modifiers[leftKey].(type) {
						case map[string]interface{}:
							newRightVars.Modifiers = rightVars.Modifiers[leftKey].(map[string]interface{})
						}
						newRightVars.Vars = rightValue.(map[string]interface{})
						comboVars := CombineVars(newLeftVars, newRightVars)
						combinedVars.Vars[leftKey] = comboVars.Vars
						combinedVars.Modifiers[leftKey] = comboVars.Modifiers
						combinedVars.Modifiers[leftKey+VarsModifiersSuffix] = rightVarsModifiers
					default:
						combinedVars.Vars[leftKey] = rightValue
						combinedVars.Modifiers[leftKey] = rightVars.Modifiers[leftKey]
						combinedVars.Modifiers[leftKey+VarsModifiersSuffix] = rightVarsModifiers
					}
				case nil:
					combinedVars.Vars[leftKey] = leftValue
					combinedVars.Modifiers[leftKey] = nil
					combinedVars.Modifiers[leftKey+VarsModifiersSuffix] = rightVarsModifiers
				default:
					combinedVars.Vars[leftKey] = rightValue
					combinedVars.Modifiers[leftKey] = nil
					combinedVars.Modifiers[leftKey+VarsModifiersSuffix] = rightVarsModifiers
				}
			}
		case leftVarsModifiers.Update:
			switch leftValue.(type) {
			case []interface{}:
				switch rightValue.(type) {
				case []interface{}:
					combinedVars.Vars[leftKey] = append(rightValue.([]interface{}), leftValue.([]interface{})...)
					combinedVars.Modifiers[leftKey+VarsModifiersSuffix] = leftVarsModifiers
					combinedVars.Modifiers[leftKey] = leftVars.Modifiers[leftKey]
				default:
					combinedVars.Vars[leftKey] = leftValue
					combinedVars.Modifiers[leftKey+VarsModifiersSuffix] = leftVarsModifiers
					combinedVars.Modifiers[leftKey] = leftVars.Modifiers[leftKey]
				}
			case map[string]interface{}:
				switch rightValue.(type) {
				case map[string]interface{}:
					newLeftVars := new(types.StackVars)
					newLeftVars.Modifiers = leftVars.Modifiers[leftKey].(map[string]interface{})
					newLeftVars.Vars = leftValue.(map[string]interface{})
					newRightVars := new(types.StackVars)
					switch rightVars.Modifiers[leftKey].(type) {
					case map[string]interface{}:
						newRightVars.Modifiers = rightVars.Modifiers[leftKey].(map[string]interface{})
						newRightVars.Vars = rightValue.(map[string]interface{})
					}
					comboVars := CombineVars(newLeftVars, newRightVars)
					combinedVars.Vars[leftKey] = comboVars.Vars
					combinedVars.Modifiers[leftKey] = comboVars.Modifiers
					combinedVars.Modifiers[leftKey+VarsModifiersSuffix] = leftVarsModifiers
				default:
					combinedVars.Vars[leftKey] = leftValue
					combinedVars.Modifiers[leftKey] = leftVars.Modifiers[leftKey]
					combinedVars.Modifiers[leftKey+VarsModifiersSuffix] = leftVarsModifiers
				}
			default:
				combinedVars.Vars[leftKey] = leftValue
				combinedVars.Modifiers[leftKey] = nil
				combinedVars.Modifiers[leftKey+VarsModifiersSuffix] = leftVarsModifiers
			}
		default:
			combinedVars.Vars[leftKey] = leftValue
			combinedVars.Modifiers[leftKey] = nil
			combinedVars.Modifiers[leftKey+VarsModifiersSuffix] = leftVarsModifiers
		}
		spew.Sprint(rightValue, rightVarsModifiers)
	}
	return
}
