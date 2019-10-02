package gedis

import (
	"encoding/json"
	"strings"
)

//Error is an error returned by a gedis server
type Error struct {
	Message  string `json:"message,omitempty"`
	Level    uint   `json:"level,omitempty"`
	Context  string `json:"context,omitempty"`
	Category string `json:"cat,omitempty"`
}

func (err Error) Error() string {
	return err.Message
}

func parseError(err error) error {
	if err == nil {
		return nil
	}

	value := strings.TrimPrefix(err.Error(), "ERR ")

	r := strings.NewReader(value)
	var gErr Error
	if jErr := json.NewDecoder(r).Decode(&gErr); jErr != nil {
		return err
	}

	return gErr
}
