package main

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"docker-auto/internal/model"
)

// JSONContainerMetrics wraps ContainerMetricsSnapshot with custom serialization
type JSONContainerMetrics model.ContainerMetricsSnapshot

// Value implements driver.Valuer interface
func (j JSONContainerMetrics) Value() (driver.Value, error) {
	if (model.ContainerMetricsSnapshot)(j) == (model.ContainerMetricsSnapshot{}) {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface
func (j *JSONContainerMetrics) Scan(value interface{}) error {
	if value == nil {
		*j = JSONContainerMetrics{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into JSONContainerMetrics", value)
	}

	return json.Unmarshal(bytes, j)
}

// JSONResourceUsage wraps ResourceUsageData with custom serialization
type JSONResourceUsage model.ResourceUsageData

// Value implements driver.Valuer interface
func (j JSONResourceUsage) Value() (driver.Value, error) {
	if (model.ResourceUsageData)(j) == (model.ResourceUsageData{}) {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface
func (j *JSONResourceUsage) Scan(value interface{}) error {
	if value == nil {
		*j = JSONResourceUsage{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into JSONResourceUsage", value)
	}

	return json.Unmarshal(bytes, j)
}

// JSONProcessInfo wraps ProcessInfo with custom serialization
type JSONProcessInfo model.ProcessInfo

// Value implements driver.Valuer interface
func (j JSONProcessInfo) Value() (driver.Value, error) {
	if (model.ProcessInfo)(j) == (model.ProcessInfo{}) {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface
func (j *JSONProcessInfo) Scan(value interface{}) error {
	if value == nil {
		*j = JSONProcessInfo{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into JSONProcessInfo", value)
	}

	return json.Unmarshal(bytes, j)
}

// JSONExtendedMetrics wraps ExtendedMetricsData with custom serialization
type JSONExtendedMetrics model.ExtendedMetricsData

// Value implements driver.Valuer interface
func (j JSONExtendedMetrics) Value() (driver.Value, error) {
	if (model.ExtendedMetricsData)(j) == (model.ExtendedMetricsData{}) {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface
func (j *JSONExtendedMetrics) Scan(value interface{}) error {
	if value == nil {
		*j = JSONExtendedMetrics{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into JSONExtendedMetrics", value)
	}

	return json.Unmarshal(bytes, j)
}

// JSONAnomalies wraps AnomaliesData with custom serialization
type JSONAnomalies model.AnomaliesData

// Value implements driver.Valuer interface
func (j JSONAnomalies) Value() (driver.Value, error) {
	if (model.AnomaliesData)(j) == (model.AnomaliesData{}) {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface
func (j *JSONAnomalies) Scan(value interface{}) error {
	if value == nil {
		*j = JSONAnomalies{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into JSONAnomalies", value)
	}

	return json.Unmarshal(bytes, j)
}

// JSONTTYSettings wraps TTYSettings with custom serialization
type JSONTTYSettings model.TTYSettings

// Value implements driver.Valuer interface
func (j JSONTTYSettings) Value() (driver.Value, error) {
	if (model.TTYSettings)(j) == (model.TTYSettings{}) {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface
func (j *JSONTTYSettings) Scan(value interface{}) error {
	if value == nil {
		*j = JSONTTYSettings{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into JSONTTYSettings", value)
	}

	return json.Unmarshal(bytes, j)
}

// JSONSessionCapabilities wraps SessionCapabilities with custom serialization
type JSONSessionCapabilities model.SessionCapabilities

// Value implements driver.Valuer interface
func (j JSONSessionCapabilities) Value() (driver.Value, error) {
	if (model.SessionCapabilities)(j) == (model.SessionCapabilities{}) {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface
func (j *JSONSessionCapabilities) Scan(value interface{}) error {
	if value == nil {
		*j = JSONSessionCapabilities{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into JSONSessionCapabilities", value)
	}

	return json.Unmarshal(bytes, j)
}

// JSONSessionHistory wraps SessionHistory with custom serialization
type JSONSessionHistory model.SessionHistory

// Value implements driver.Valuer interface
func (j JSONSessionHistory) Value() (driver.Value, error) {
	if (model.SessionHistory)(j) == (model.SessionHistory{}) {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface
func (j *JSONSessionHistory) Scan(value interface{}) error {
	if value == nil {
		*j = JSONSessionHistory{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into JSONSessionHistory", value)
	}

	return json.Unmarshal(bytes, j)
}