# Performance Tests

Performance tests measure system performance under load and identify bottlenecks.

## Characteristics

- **Benchmark critical code paths**
- **Load testing for API endpoints**
- **Memory and CPU profiling**
- **Focus on scalability and performance**

## Running Performance Tests

```bash
# Run benchmark tests
go test -bench=. ./tests/performance/... -benchmem

# Run with specific benchmark
go test -bench=BenchmarkAssistantReply ./tests/performance/... -benchmem

# Run load tests
go test ./tests/performance/load/... -v

# Generate CPU profile
go test -bench=. -cpuprofile=cpu.prof ./tests/performance/...
```

## Test Types

- **Benchmark tests** for critical functions
- **Load tests** for API endpoints
- **Memory profiling** for leak detection
- **Concurrency testing** for race conditions

## Test Structure

- Use Go's built-in benchmarking
- Test under realistic load conditions
- Profile memory and CPU usage
- Measure response times and throughput
