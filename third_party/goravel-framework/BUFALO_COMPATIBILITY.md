# BUFALO compatibility snapshot

This directory contains github.com/goravel/framework v1.18.0, with its original LICENSE. Upstream test files and CI workflows were excluded; application runtime files otherwise retain their upstream contents except for the two files below.

OpenTelemetry log v0.21.0 fixes known vulnerabilities and uses `attribute.Value` and `attribute.KeyValue` for structured values. Goravel v1.18.0 still uses their previous log API equivalents. BUFALO adapts only:

- `telemetry/instrumentation/log/mapper.go`: value constructors and nested collections use the attribute API; byte slices remain byte slices and dynamic map keys convert to `attribute.Key`.
- `telemetry/instrumentation/log/handler.go`: structured log attributes use the attribute key/value type.

`telemetry/instrumentation/log/bufalo_compat_test.go` verifies structured values, bytes, nil values and unsigned overflow handling. Run it from this directory with `go test ./telemetry/instrumentation/log` using the project's patched Go toolchain. The application module selects this snapshot through its `replace` directive.

When updating Goravel, compare these changes with upstream, keep the patched OpenTelemetry versions, rerun the compatibility test and the application acceptance suite, and repeat binary vulnerability scanning. Remove the replacement when upstream supports the patched API without adaptation. Do not modify generated mocks.
