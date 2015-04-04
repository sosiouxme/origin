package log

import (
	"encoding/json"
	"fmt"
)

type jsonLogger struct {
	logStarted  bool
	logFinished bool
}

func (j *jsonLogger) Write(l Level, msg Msg) {
	if j.logStarted {
		fmt.Println(",")
	} else {
		fmt.Println("[")
	}
	j.logStarted = true
	msg["level"] = l.Name
	b, _ := json.MarshalIndent(msg, "  ", "  ")
	fmt.Print("  " + string(b))
}
func (j *jsonLogger) Finish() {
	if j.logStarted {
		fmt.Println("\n]")
	} else if !j.logFinished {
		fmt.Println("[]")
	}
	j.logFinished = true
}
