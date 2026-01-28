package api

import (
	"encoding/json"
	"errors"
)

// decodeJSON decodes JSON into dst.
func decodeJSON(raw json.RawMessage, dst any) error {
	if len(raw) == 0 || string(raw) == "null" {
		return errors.New("missing data")
	}

	if err := json.Unmarshal(raw, dst); err != nil {
		return err
	}
	return nil
}

// decodeString decodes JSON into a string.
func decodeString(raw json.RawMessage) (string, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", err
	}
	return value, nil
}

// decodeBool decodes JSON into a bool.
func decodeBool(raw json.RawMessage) (bool, error) {
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		return false, err
	}
	return value, nil
}
