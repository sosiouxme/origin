package log

import (
	"bytes"
	"errors"
	"fmt"
	ct "github.com/daviddengcn/go-colortext"
	"strings"
	"text/template"
)

type Level struct {
	Level  int
	Name   string
	Prefix string
	Color  ct.Color
	Bright bool
}

//throwing type safety and method signatures out the window:
type Msg map[string]interface{}

/* a Msg can be expected to have the following entries:
 * "id": an identifier unique to the message being logged, intended for json/yaml output
 *       so that automation can recognize specific messages without trying to parse them.
 * "text": human-readable message text
 * "tmpl": a template string as understood by text/template that can use any of the other
 *         entries in this Msg as inputs. This is removed, evaluated, and the result is
 *         placed in "text". If there is an error during evaluation, the error is placed
 *         in "templateErr", the original id of the message is stored in "templateId",
 *         and the Msg id is changed to "tmplErr". Of course, this should never happen
 *         if there are no mistakes in the calling code.
 */

var (
	ErrorLevel  = Level{0, "error", "ERROR: ", ct.Red, true}   // Something is definitely wrong
	WarnLevel   = Level{1, "warn", "WARN:  ", ct.Yellow, true} // Likely to be an issue but maybe not
	InfoLevel   = Level{2, "info", "Info:  ", ct.None, false}  // Just informational
	NoticeLevel = Level{2, "note", "[Note] ", ct.White, false} // Introductory / summary
	DebugLevel  = Level{3, "debug", "debug: ", ct.None, false} // Extra verbose
)

var current Level = InfoLevel // default
var warningsSeen int = 0
var errorsSeen int = 0

func SetLevel(level int) {
	switch level {
	case 0:
		current = ErrorLevel
	case 1:
		current = WarnLevel
	case 2:
		current = InfoLevel
	default:
		current = DebugLevel
	}
}

//
// Deal with different log formats
//
type loggerType interface {
	Write(Level, Msg)
	Finish()
}

var logger loggerType = &textLogger{} // default to human user
func SetLogFormat(format string) error {
	logger = &textLogger{} // default
	switch format {
	case "json":
		logger = &jsonLogger{}
	case "yaml":
		logger = &yamlLogger{}
	case "text":
	default:
		return errors.New("Output format must be one of: text, json, yaml")
	}
	return nil
}

// Provide a summary at the end
func Summary() {
	Notice("summary", "\nSummary of diagnostics execution:\n")
	if warningsSeen > 0 {
		Noticem("sumWarn", Msg{"tmpl": "Warnings seen: {{.num}}", "num": warningsSeen})
	}
	if errorsSeen > 0 {
		Noticem("sumErr", Msg{"tmpl": "Errors seen: {{.num}}", "num": errorsSeen})
	}
	if warningsSeen == 0 && errorsSeen == 0 {
		Notice("sumNone", "Completed with no errors or warnings seen.")
	}
}

func Log(l Level, id string, msg Msg) {
	if l.Level > current.Level {
		return
	}
	msg["id"] = id // TODO: use to retrieve template from elsewhere
	// if given a template, convert it to text
	if tmpl, exists := msg["tmpl"]; exists {
		var buff bytes.Buffer
		if tmplString, assertion := tmpl.(string); !assertion {
			msg["templateErr"] = fmt.Sprintf("Invalid template type: %T", tmpl)
			msg["templateId"] = id
			msg["id"] = "tmplErr"
		} else {
			parsedTmpl, err := template.New(id).Parse(tmplString)
			if err != nil {
				msg["templateErr"] = err.Error()
				msg["templateId"] = id
				msg["id"] = "tmplErr"
			} else if err = parsedTmpl.Execute(&buff, msg); err != nil {
				msg["templateErr"] = err.Error()
				msg["templateId"] = id
				msg["id"] = "tmplErr"
			} else {
				msg["text"] = buff.String()
				delete(msg, "tmpl")
			}
		}
	}
	if l.Level == ErrorLevel.Level {
		errorsSeen += 1
	} else if l.Level == WarnLevel.Level {
		warningsSeen += 1
	}
	logger.Write(l, msg)
}

// Convenience functions
func Error(id string, text string) {
	Log(ErrorLevel, id, Msg{"text": text})
}
func Errorf(id string, msg string, a ...interface{}) {
	Error(id, fmt.Sprintf(msg, a...))
}
func Errorm(id string, msg Msg) {
	Log(ErrorLevel, id, msg)
}
func Warn(id string, text string) {
	Log(WarnLevel, id, Msg{"text": text})
}
func Warnf(id string, msg string, a ...interface{}) {
	Warn(id, fmt.Sprintf(msg, a...))
}
func Warnm(id string, msg Msg) {
	Log(WarnLevel, id, msg)
}
func Info(id string, text string) {
	Log(InfoLevel, id, Msg{"text": text})
}
func Infof(id string, msg string, a ...interface{}) {
	Info(id, fmt.Sprintf(msg, a...))
}
func Infom(id string, msg Msg) {
	Log(InfoLevel, id, msg)
}
func Notice(id string, text string) {
	Log(NoticeLevel, id, Msg{"text": text})
}
func Noticef(id string, msg string, a ...interface{}) {
	Notice(id, fmt.Sprintf(msg, a...))
}
func Noticem(id string, msg Msg) {
	Log(NoticeLevel, id, msg)
}
func Debug(id string, text string) {
	Log(DebugLevel, id, Msg{"text": text})
}
func Debugf(id string, msg string, a ...interface{}) {
	Debug(id, fmt.Sprintf(msg, a...))
}
func Debugm(id string, msg Msg) {
	Log(DebugLevel, id, msg)
}

// turn excess lines into [...]
func LimitLines(msg string, n int) string {
	lines := strings.SplitN(msg, "\n", n+1)
	if len(lines) == n+1 {
		lines[n] = "[...]"
	}
	return strings.Join(lines, "\n")
}

func Finish() {
	logger.Finish()
}
