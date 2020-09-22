package consts

// MESSAGES
const (
	MessageBadStack                  = "Bad stack"
	MessageBadStackErr               = "Bad stack: %s"
	MessageBadStackUnsupportedAPI    = "Bad stack. Unsupported API"
	MessageChanged                   = "Changed"
	MessageLibsBadItem               = "Bad lib item"
	MessageLibsGitBadPathInRepo      = "Bad path %s in git repo %s"
	MessageLibsParseAndInit          = "Parse and init lib item: %s"
	MessagePathNotFoundInSearchPaths = "Path %s not found. Search paths:\n%s"
	MessagesReadingStackFrom         = "Reading stack from"
	MessageVarsBadVarName            = "Bad var name! Probably unexpected behavior"
	MessageVarsDoubleDefinition      = "Var double definition"
	MessageVarsSimplyfy              = "Simplyfy var name to <%s> Probably unexpected behavior"
)

// ExitCodes
const (
	ExitCodeOK = iota
)

// other
const (
	GitCloneDir          = ".gitclone"
	StackDefaultFileName = "stack"
)
