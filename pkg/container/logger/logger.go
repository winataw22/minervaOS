package logger

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
