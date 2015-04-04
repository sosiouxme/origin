package log

import (
	"fmt"
	"gopkg.in/yaml.v2"
)

type yamlLogger struct {
	logStarted bool
}

func (y *yamlLogger) Write(l Level, msg Msg) {
	msg["level"] = l.Name
	b, _ := yaml.Marshal(&msg)
	fmt.Println("---\n" + string(b))
}
func (y *yamlLogger) Finish() {}
