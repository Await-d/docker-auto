package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Value implements driver.Valuer interface for ContainerMetricsSnapshot
func (c ContainerMetricsSnapshot) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan implements sql.Scanner interface for ContainerMetricsSnapshot
func (c *ContainerMetricsSnapshot) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into ContainerMetricsSnapshot", value)
	}

	return json.Unmarshal(bytes, c)
}

// Value implements driver.Valuer interface for ResourceUsageData
func (r ResourceUsageData) Value() (driver.Value, error) {
	return json.Marshal(r)
}

// Scan implements sql.Scanner interface for ResourceUsageData
func (r *ResourceUsageData) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into ResourceUsageData", value)
	}

	return json.Unmarshal(bytes, r)
}

// Value implements driver.Valuer interface for ProcessInfo
func (p ProcessInfo) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// Scan implements sql.Scanner interface for ProcessInfo
func (p *ProcessInfo) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into ProcessInfo", value)
	}

	return json.Unmarshal(bytes, p)
}

// Value implements driver.Valuer interface for ExtendedMetricsData
func (e ExtendedMetricsData) Value() (driver.Value, error) {
	return json.Marshal(e)
}

// Scan implements sql.Scanner interface for ExtendedMetricsData
func (e *ExtendedMetricsData) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into ExtendedMetricsData", value)
	}

	return json.Unmarshal(bytes, e)
}

// Value implements driver.Valuer interface for AnomaliesData
func (a AnomaliesData) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan implements sql.Scanner interface for AnomaliesData
func (a *AnomaliesData) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into AnomaliesData", value)
	}

	return json.Unmarshal(bytes, a)
}

// Value implements driver.Valuer interface for TTYSettings
func (t TTYSettings) Value() (driver.Value, error) {
	return json.Marshal(t)
}

// Scan implements sql.Scanner interface for TTYSettings
func (t *TTYSettings) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into TTYSettings", value)
	}

	return json.Unmarshal(bytes, t)
}

// Value implements driver.Valuer interface for SessionCapabilities
func (s SessionCapabilities) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan implements sql.Scanner interface for SessionCapabilities
func (s *SessionCapabilities) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into SessionCapabilities", value)
	}

	return json.Unmarshal(bytes, s)
}

// Value implements driver.Valuer interface for SessionHistory
func (s SessionHistory) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan implements sql.Scanner interface for SessionHistory
func (s *SessionHistory) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into SessionHistory", value)
	}

	return json.Unmarshal(bytes, s)
}