package tengo2lua

// Options represents a set of options for Transpiler.
type Options struct {
	// EnableGlobalScope creates global variables if it's set to true.
	// If not, global variables in Tengo code will be transpiled as local
	// variables in Lua.
	EnableGlobalScope bool

	// Indent string is added whenever the block level increases.
	Indent string
}

// DefaultOptions creates a default option for Transpiler.
func DefaultOptions() *Options {
	return &Options{
		EnableGlobalScope: false,
		Indent:            "  ",
	}
}
