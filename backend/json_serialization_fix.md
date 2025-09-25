# JSON Serialization Fix for SQLite Compatibility

## Issue Identified

The Container model uses complex struct pointers with GORM's `type:jsonb` tag, which works for PostgreSQL but fails in SQLite with the error:

```
sql: Scan error on column index 30, name "last_metrics": unsupported Scan,
storing driver.Value type string into type *model.ContainerMetricsSnapshot
```

## Root Cause Analysis

1. **PostgreSQL vs SQLite**: PostgreSQL has native JSONB support, SQLite stores JSON as TEXT
2. **GORM Scanning**: GORM expects struct types but receives string values from SQLite
3. **Type Mismatch**: Complex structs like `ContainerMetricsSnapshot` need custom serialization

## Solution Strategy

### Option 1: Custom Scanner/Valuer Implementation (Recommended)
Implement `driver.Valuer` and `sql.Scanner` interfaces for JSON types:

```go
func (c *ContainerMetricsSnapshot) Value() (driver.Value, error) {
    if c == nil {
        return nil, nil
    }
    return json.Marshal(c)
}

func (c *ContainerMetricsSnapshot) Scan(value interface{}) error {
    if value == nil {
        return nil
    }

    bytes, ok := value.([]byte)
    if !ok {
        return errors.New("cannot scan ContainerMetricsSnapshot")
    }

    return json.Unmarshal(bytes, c)
}
```

### Option 2: Database-Specific Schema (Alternative)
Use different field types based on database type:
- SQLite: Store as TEXT with manual JSON handling
- PostgreSQL: Use native JSONB support

## Implementation Plan

1. **Add Scanner/Valuer interfaces** to complex struct types
2. **Update Container model** field tags if needed
3. **Test with both SQLite and PostgreSQL**
4. **Validate CRUD operations** work correctly
5. **Performance test** JSON serialization overhead

## Testing Strategy

The validation test revealed:
- ✅ Database connection and migrations work
- ✅ String-based JSON fields work (Labels, Environment, etc.)
- ❌ Complex struct JSON fields fail (LastMetrics, ResourceUsage, ProcessInfo)
- ✅ Most other functionality is operational

## Priority

**HIGH PRIORITY** - This blocks Container model functionality which is core to Docker integration.

## Files to Modify

1. `internal/model/container.go` - Add Scanner/Valuer methods
2. `internal/model/monitoring_metrics.go` - Add Scanner/Valuer for ExtendedMetrics
3. `internal/model/terminal_session.go` - Add Scanner/Valuer for complex JSON fields

## Success Criteria

- All Container CRUD operations work with SQLite
- JSON fields serialize/deserialize correctly
- Performance remains acceptable (<10ms for typical operations)
- PostgreSQL compatibility maintained