package appconfig

// AppConfig type
type AppConfig struct {
	TagPatterns *[]string `json:"tags,omitempty"`
	LogFormat   *string   `json:"logformat,omitempty"`
	CLIValues   *[]string `json:"clivalues,omitempty"`
	VarFiles    *[]string `json:"filevalues,omitempty"`
	Workspace   *string   `json:"workspace,omitempty"`
	Verbosity   *int      `json:"verbosity,omitempty"`
}
