package logger

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

type logEntry struct {
	Timestamp string                 `json:"ts"`
	Level     string                 `json:"level"`
	Message   string                 `json:"msg"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

var output = log.New(os.Stdout, "", 0)

func emit(level, msg string, extra map[string]interface{}) {
	entry := logEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Message:   msg,
		Extra:     extra,
	}
	data, _ := json.Marshal(entry)
	output.Println(string(data))
}

func Info(msg string, extra map[string]interface{}) {
	emit("info", msg, extra)
}

func Warn(msg string, extra map[string]interface{}) {
	emit("warn", msg, extra)
}

func Error(msg string, extra map[string]interface{}) {
	emit("error", msg, extra)
}
