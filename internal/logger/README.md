# Benchmarks
```
goos: darwin
goarch: arm64
pkg: poster/internal/logger
cpu: Apple M4
BenchmarkLogInfo-10                      1984448               596.3 ns/op           825 B/op          8 allocs/op
BenchmarkLogDebug-10                     1932962               605.4 ns/op           857 B/op          8 allocs/op
BenchmarkLogError-10                     1998835               603.4 ns/op           857 B/op          8 allocs/op
BenchmarkLogWithOneField-10              1341598               888.0 ns/op          1274 B/op         10 allocs/op
BenchmarkLogWithTenFields-10              650427              1778 ns/op            2014 B/op         12 allocs/op
BenchmarkLogWithFieldsMerge-10            824749              1433 ns/op            1435 B/op         10 allocs/op
BenchmarkLogParallel-10                   979420              1177 ns/op            1276 B/op         10 allocs/op
BenchmarkLogWithCallerInfo-10            2066271               578.1 ns/op           857 B/op          8 allocs/op
BenchmarkLogNoOutput-10                  4356643               277.1 ns/op           296 B/op          3 allocs/op
```