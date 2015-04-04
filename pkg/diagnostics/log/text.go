package log

import (
	"fmt"
	ct "github.com/daviddengcn/go-colortext"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"runtime"
	"strings"
)

var ttyOutput bool = true   // don't output colors to non-tty
var lastNewline bool = true // keep track of newline separation

func init() {
	if runtime.GOOS == "linux" && !terminal.IsTerminal(int(os.Stdout.Fd())) {
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
