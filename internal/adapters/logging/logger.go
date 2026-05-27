package logging

import (
	"fmt"
	"os"
)

type Level string

const (
	Silent Level = "silent"
	Warn   Level = "warn"
	Debug  Level = "debug"
)

type StdLogger struct {
	level Level
}

func New(level string) StdLogger {
	return StdLogger{level: Level(level)}
}

func (l StdLogger) Debug(msg string) {
	if l.level == Debug {
		fmt.Fprintln(os.Stderr, "DEBUG:", msg)
	}
}

func (l StdLogger) Warn(msg string) {
	if l.level == Warn || l.level == Debug {
		fmt.Fprintln(os.Stderr, "WARN:", msg)
	}
}
