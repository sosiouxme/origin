package log

import (
	"fmt"
	ct "github.com/daviddengcn/go-colortext"
	"github.com/openshift/origin/pkg/cmd/util"
	"os"
	"strings"
)

var ttyOutput bool = true   // usually want color; but do not output colors to non-tty
var lastNewline bool = true // keep track of newline separation

func init() {
	if !util.IsTerminal(os.Stdout) {
		// don't want color sequences in redirected output (logs, "less", etc.)
		ttyOutput = false
	}
}

type textLogger struct{}

func (t *textLogger) Write(l Level, msg Msg) {
	if ttyOutput {
		ct.ChangeColor(l.Color, l.Bright, ct.None, false)
	}
	text := strings.TrimSpace(fmt.Sprintf("%v", msg["text"]))
	if strings.Contains(text, "\n") { // separate multiline comments with newlines
		if !lastNewline {
			fmt.Println() // separate from previous one-line log msg
		}
		text = text + "\n"
		lastNewline = true
	} else {
		lastNewline = false
	}
	fmt.Println(l.Prefix + strings.Replace(text, "\n", "\n       ", -1))
	if ttyOutput {
		ct.ResetColor()
	}
}
func (t *textLogger) Finish() {}
