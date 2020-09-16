package misc

import (
	"runtime/debug"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/kruglovmax/stack/pkg/log"
)

/*
DeleteKey func
*/
func DeleteKey(vars *interface{}, key string) {
	key = strings.TrimRight(key, "~")
	if len(key) == 0 {
		log.Logger.Trace().
			Msg(spew.Sdump(vars))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg("The key cannot be \"+\"")
	}
	switch (*vars).(type) {
	case map[string]interface{}:
		for k := range (*vars).(map[string]interface{}) {
			if strings.TrimRight(k, "~") == key {
				delete((*vars).(map[string]interface{}), k)
			}
		}
	default:
		log.Logger.Trace().
			Msg(spew.Sdump(vars))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg("Object is not map")
	}
}

// GetRealVars func
func GetRealVars(vars interface{}) interface{} {
	// var result interface{}
	switch vars.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, v := range vars.(map[string]interface{}) {
			result[strings.TrimRight(k, "~")] = GetRealVars(v)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(vars.([]interface{})))
		for k, v := range vars.([]interface{}) {
			result[k] = GetRealVars(v)
		}
		return result
	default:
		return vars
	}
}

/*
GetVar func
*/
func GetVar(vars interface{}, key string) (interface{}, bool) {
	key = strings.TrimRight(key, "~")
	if len(key) == 0 {
		log.Logger.Trace().
			Msg(spew.Sdump(vars))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg("The key cannot be \"+\"")
	}
	switch vars.(type) {
	case map[string]interface{}:
		for k, v := range vars.(map[string]interface{}) {
			if strings.TrimRight(k, "~") == key {
				return v, true
			}
		}
	default:
		log.Logger.Trace().
			Msg(spew.Sdump(vars))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg("Object is not map")
	}
	return nil, false
}

/*
GetRealKeyName func
*/
func GetRealKeyName(vars interface{}, key string) string {
	key = strings.TrimRight(key, "~")
	if len(key) == 0 {
		log.Logger.Trace().
			Msg(spew.Sdump(vars))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg("The key cannot be \"+\"")
	}

	switch vars.(type) {
	case map[string]interface{}:
		for k := range vars.(map[string]interface{}) {
			if strings.TrimRight(k, "~") == key {
				return k
			}
		}
	default:
		log.Logger.Trace().
			Msg(spew.Sdump(vars))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg("Object is not map")
	}
	return ""
}

// SetRealKeys sets + recursively to map keys
func SetRealKeys(vars interface{}) interface{} {
	switch vars.(type) {
	case map[string]interface{}:
		for key, value := range vars.(map[string]interface{}) {
			switch key[len(key)-1] {
			case '+':
				switch value.(type) {
				case map[string]interface{}:
					abc := value.(map[string]interface{})
					for k, v := range value.(map[string]interface{}) {
						if k[len(k)-1] != '+' {
							abc[k+"~"] = v
							delete(abc, k)
						}
					}
					vars.(map[string]interface{})[key] = abc
				}
			}
		}
		for key, value := range vars.(map[string]interface{}) {
			switch value.(type) {
			case map[string]interface{}:
				vars.(map[string]interface{})[key] = SetRealKeys(vars.(map[string]interface{})[key])
			}
		}
		return vars
	default:
		return nil
	}
}

// CombineVars func
func CombineVars(leftVars, rightVars interface{}) map[string]interface{} {
	leftVars = SetRealKeys(leftVars)
	rightVars = SetRealKeys(rightVars)
	combinedVars := make(map[string]interface{})

	if leftVars != nil && rightVars != nil {
		for leftKey, leftValue := range leftVars.(map[string]interface{}) {
			switch leftKey[len(leftKey)-1] {
			case '+':
				if valueTmp, ok := GetVar(rightVars, leftKey); ok {
					keyTmp := GetRealKeyName(rightVars, leftKey)
					switch valueTmp.(type) {
					case map[string]interface{}:
						switch leftValue.(type) {
						case map[string]interface{}:
							leftVar, _ := GetVar(leftVars, leftKey)
							rightVar, _ := GetVar(rightVars, leftKey)
							combinedVars[keyTmp] = CombineVars(leftVar.(map[string]interface{}), rightVar.(map[string]interface{}))
							DeleteKey(&rightVars, keyTmp)
						default:
							combinedVars[keyTmp] = valueTmp
							DeleteKey(&rightVars, keyTmp)
						}
					case []interface{}:
						switch leftValue.(type) {
						case []interface{}:
							if k := strings.TrimRight(leftKey, "~"); k+"~~" == leftKey {
								combinedVars[keyTmp] = append(leftValue.([]interface{}), valueTmp.([]interface{})...)
								DeleteKey(&rightVars, keyTmp)
							} else {
								combinedVars[keyTmp] = valueTmp
								DeleteKey(&rightVars, keyTmp)
							}
						default:
							combinedVars[keyTmp] = valueTmp
							DeleteKey(&rightVars, keyTmp)
						}
					default:
						combinedVars[keyTmp] = valueTmp
						DeleteKey(&rightVars, keyTmp)
					}
				} else {
					combinedVars[leftKey] = leftValue
				}
			default:
				switch leftValue.(type) {
				case map[string]interface{}:
					if v, _ := GetVar(rightVars, leftKey); v == nil {
						combinedVars[leftKey] = leftValue
					} else {
						switch cVar, _ := GetVar(rightVars, leftKey); cVar.(type) {
						case map[string]interface{}:
							pVar, _ := GetVar(leftVars, leftKey)
							combinedVars[leftKey] = CombineVars(pVar.(map[string]interface{}), cVar.(map[string]interface{}))
							DeleteKey(&rightVars, leftKey)
						default:
							combinedVars[leftKey] = leftValue
							DeleteKey(&rightVars, leftKey)
						}
					}
				default:
					combinedVars[leftKey] = leftValue
					DeleteKey(&rightVars, leftKey)
				}
			}
		}
		for key, value := range rightVars.(map[string]interface{}) {
			if value != nil {
				combinedVars[key] = value
			}
		}
	} else if rightVars != nil {
		combinedVars = rightVars.(map[string]interface{})
	} else if leftVars != nil {
		combinedVars = leftVars.(map[string]interface{})
	}

	return combinedVars
}
