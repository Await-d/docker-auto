package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSONMap represents a JSON object stored as a string in the database
type JSONMap map[string]interface{}

// Value implements the driver.Valuer interface for database storage
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return "{}", nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for database retrieval
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONMap)
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into JSONMap", value)
	}

	return json.Unmarshal(bytes, j)
}

// String returns the JSON string representation
func (j JSONMap) String() string {
	if j == nil {
		return "{}"
	}
	bytes, _ := json.Marshal(j)
	return string(bytes)
}

// Set allows setting values in the JSON map
func (j JSONMap) Set(key string, value interface{}) {
	j[key] = value
}

// Get retrieves a value from the JSON map
func (j JSONMap) Get(key string) interface{} {
	return j[key]
}