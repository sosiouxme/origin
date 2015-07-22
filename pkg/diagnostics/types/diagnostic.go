package types

// This needed to be separate from other types to avoid import cycle
// diagnostic -> discovery -> types

import (
	"github.com/openshift/origin/pkg/diagnostics/log"
)

type Diagnostic interface {
	Description() string
	CanRun() (canRun bool, reason error)
	Check() *DiagnosticResult
}

type DiagnosticResult struct {
	failure  bool
	logs     []log.Message
	warnings []error
	errors   []error
}

func NewDiagnosticResult() *DiagnosticResult {
	return &DiagnosticResult{false, nil, nil, nil}
}

func (r *DiagnosticResult) Complete() *DiagnosticResult {
	if r.errors == nil {
		r.errors = make([]error, 0)
	}
	if r.warnings == nil {
		r.warnings = make([]error, 0)
	}
	if r.logs == nil {
		r.logs = make([]log.Message, 0)
	}
	return r
}

func (r *DiagnosticResult) Log(msg ...log.Message) *DiagnosticResult {
	if r.logs == nil {
		r.logs = make([]log.Message, 0)
	}
	r.logs = append(r.logs, msg...)
	return r
}

func (r *DiagnosticResult) Logs() []log.Message {
	if r.logs == nil {
		return make([]log.Message, 0)
	}
	return r.logs
}

func (r *DiagnosticResult) Warn(warn ...error) *DiagnosticResult {
	if r.warnings == nil {
		r.warnings = make([]error, 0)
	}
	r.warnings = append(r.warnings, warn...)
	return r
}

func (r *DiagnosticResult) Warnings() []error {
	if r.warnings == nil {
		return make([]error, 0)
	}
	return r.warnings
}

func (r *DiagnosticResult) Error(err ...error) *DiagnosticResult {
	if r.errors == nil {
		r.errors = make([]error, 0)
	}
	r.failure = true
	r.errors = append(r.errors, err...)
	return r
}

func (r *DiagnosticResult) Errors() []error {
	if r.errors == nil {
		return make([]error, 0)
	}
	return r.errors
}

func (r *DiagnosticResult) Append(r2 *DiagnosticResult) *DiagnosticResult {
	r.Complete()
	r2.Complete()
	r.logs = append(r.logs, r2.logs...)
	r.warnings = append(r.warnings, r2.warnings...)
	r.errors = append(r.errors, r2.errors...)
	r.failure = r.failure || r2.failure
	return r
}
