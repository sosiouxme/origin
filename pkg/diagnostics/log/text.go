package log

import (
	"fmt"
	ct "github.com/daviddengcn/go-colortext"
	"os"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

var ttyOutput bool = true   // usually want color; but do not output colors to non-tty
var lastNewline bool = true // keep track of newline separation

func init() {
	if runtime.GOOS == "linux" {
		// embed linux-specific https://godoc.org/golang.org/x/crypto/ssh/terminal#IsTerminal
		var termios syscall.Termios
		var ioctlReadTermios = uintptr(0x5401) // syscall.TCGETS
		_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(int(os.Stdout.Fd())), ioctlReadTermios, uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
		if err != 0 { // don't want color sequences in redirected output (logs, "less", etc.)
			ttyOutput = false
		}
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
