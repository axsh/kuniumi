package kuniumi

// ParamDef defines a parameter with its name and description.
type ParamDef struct {
	Name string
	Desc string
}

// Param creates a new ParamDef.
func Param(name, desc string) ParamDef {
	return ParamDef{
		Name: name,
		Desc: desc,
	}
}

// WithParams returns a FuncOption that associates descriptions with function parameters.
func WithParams(params ...ParamDef) FuncOption {
	return func(rf *RegisteredFunc) {
		rf.paramDefs = params
	}
}

// WithReturns returns a FuncOption that specifies the description for the function return value.
func WithReturns(desc string) FuncOption {
	return func(rf *RegisteredFunc) {
		rf.returnDesc = desc
	}
}
