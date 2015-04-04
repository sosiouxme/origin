package types

type DiagnosticCondition func(env *Environment) (skip bool, reason string)

type Diagnostic struct {
	Description string
	Condition   DiagnosticCondition
	Run         func(env *Environment)
}
