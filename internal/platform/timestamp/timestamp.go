package timestamp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

const wireLayout = time.RFC3339Nano

type Timestamp struct {
	time.Time
}

func New(value time.Time) Timestamp {
	return Timestamp{Time: value.UTC()}
}

func Parse(value string) (Timestamp, error) {
	parsed, err := time.Parse(wireLayout, value)
	if err != nil {
		return Timestamp{}, fmt.Errorf("parse timestamp: %w", err)
	}
	return New(parsed), nil
}

func (t Timestamp) String() string {
	return t.UTC().Format(wireLayout)
}

func (t Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *Timestamp) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return fmt.Errorf("timestamp cannot be null")
	}

	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("timestamp must be an RFC3339 string: %w", err)
	}
	parsed, err := Parse(value)
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}
