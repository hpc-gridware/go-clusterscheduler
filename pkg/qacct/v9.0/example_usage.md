# qacct Interface Usage Examples

The qacct interface has been redesigned to provide a clean, type-safe API while maintaining backward compatibility.

## Backward Compatibility

Existing code continues to work unchanged:

```go
// Existing usage - unchanged
jobs, err := qacct.ShowJobDetails([]int64{12345})

// Direct CLI access - unchanged  
output, err := qacct.NativeSpecification([]string{"-g", "users", "-d", "7"})

// Help method - now properly implemented
help, err := qacct.ShowHelp()
```

## New Builder Pattern API

### Summary Usage Queries

```go
// Get summary usage for last 7 days for developers group
summary, err := qacct.Summary().
    LastDays(7).
    Group("developers").
    Execute()

// Get summary usage for specific time range and department
summary, err := qacct.Summary().
    BeginTime("2024-01-01").
    EndTime("2024-01-31").
    Department("engineering").
    Execute()

// Get summary for specific owner and project
summary, err := qacct.Summary().
    Owner("alice").
    Project("research").
    Execute()
```

### Advanced Job Detail Queries

```go
// Get job details with filtering
jobs, err := qacct.Jobs().
    Owner("bob").
    Queue("all.q").
    LastDays(1).
    Execute()

// Get jobs matching pattern from specific project
jobs, err := qacct.Jobs().
    JobPattern("test-*").
    Project("research").
    Execute()

// Get task details for specific job and range
jobs, err := qacct.Jobs().
    Tasks("123", "1-10").
    Host("master").
    Execute()
```

## Output Formats

### Summary Output (Usage struct)
```go
type Usage struct {
    WallClock  float64 `json:"wallclock"`
    UserTime   float64 `json:"utime"`
    SystemTime float64 `json:"stime"`
    CPU        float64 `json:"cpu"`
    Memory     float64 `json:"mem"`
    IO         float64 `json:"io"`
    IOWait     float64 `json:"iow"`
    MaxVMem    float64 `json:"maxvmem"`
    MaxRSS     float64 `json:"maxrss"`
}
```

### Job Detail Output ([]JobDetail)
Detailed job information including:
- Job metadata (owner, queue, project, etc.)
- Resource usage statistics
- Start/end times
- Exit status

## Design Benefits

1. **Backward Compatibility**: `ShowJobDetails()` and `NativeSpecification()` work exactly as before
2. **Type Safety**: Structured options instead of raw CLI strings
3. **Fluent Interface**: Chain filters naturally with method calls
4. **Honest Interface**: Only exposes working functionality (removed "not implemented" methods)
5. **qacct Flexibility**: Handles qacct's complex option combinations properly