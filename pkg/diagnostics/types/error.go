package types

import (
	"fmt"

	"github.com/openshift/origin/pkg/diagnostics/log"
)

type DiagnosticError struct {
	ID          string
	Explanation string
	Cause       error

	LogMessage *log.Message
}

func NewDiagnosticError(id, explanation string, cause error) DiagnosticError {
	return DiagnosticError{id, explanation, cause, nil}
}

func NewDiagnosticErrorFromTemplate(id, template string, templateData interface{}) DiagnosticError {
	return DiagnosticError{id, "", nil,
		&log.Message{
			ID:           id,
			Template:     template,
			TemplateData: templateData,
		},
	}
}

func (e DiagnosticError) Error() string {
	if e.Cause != nil {
		return e.Cause.Error()
	}

	if e.LogMessage != nil {
		return fmt.Sprintf("%v", e.LogMessage)
	}

	return e.Explanation
}

func IsDiagnosticError(e error) bool {
	_, ok := e.(DiagnosticError)
	return ok
}

// is the error a diagnostics error that matches the given ID?
func MatchesDiagError(err error, id string) bool {
	if derr, ok := err.(DiagnosticError); ok && derr.ID == id {
		return true
	}
	return false
}
