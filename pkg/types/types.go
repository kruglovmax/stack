package types

// Config interface
type Config interface {
	ToMap() map[string]interface{}
}
