package logger

import (
	"encoding/json"
	"os"
)

// RedisType defines redis logger type name
const RedisType = "redis"

// FileType defines file logger type name
const FileType = "file"

// ConsoleType defines console logger type name
const ConsoleType = "console"

// Logs defines a custom backend with variable settings
type Logs struct {
	Type string    `json:"type"`
	Data LogsRedis `json:"data"`
}

// LogsRedis defines how to connect a redis logs backend
type LogsRedis struct {
	// Stdout is the redis url for stdout (redis://host/channel)
	Stdout string `json:"stdout"`

	// Stderr is the redis url for stderr (redis://host/channel)
	Stderr string `json:"stderr"`
}

// Serialize dumps logs array into a json file
func Serialize(path string, logs []Logs) error {
	data, err := json.Marshal(logs)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Deserialize reads json from disks and returns []Logs
func Deserialize(path string) ([]Logs, error) {
	logs := []Logs{}

	data, err := os.ReadFile(path)
	if err != nil {
		return logs, err
	}

	if err := json.Unmarshal(data, &logs); err != nil {
		return logs, err
	}

	return logs, nil
}
