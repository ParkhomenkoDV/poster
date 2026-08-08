# Benchmarks
```
goos: darwin
goarch: arm64
pkg: poster/internal/logger
cpu: Apple M4
BenchmarkLogInfo-10                      2008018               566.9 ns/op           825 B/op          8 allocs/op
BenchmarkLogDebug-10                     2083879               571.9 ns/op           857 B/op          8 allocs/op
BenchmarkLogError-10                     2104670               569.5 ns/op           857 B/op          8 allocs/op
BenchmarkLogWithOneField-10              1411303               852.4 ns/op          1274 B/op         10 allocs/op
BenchmarkLogWithFiveFields-10             987403              1167 ns/op            1435 B/op         10 allocs/op
BenchmarkLogWithTenFields-10              647070              1788 ns/op            2308 B/op         13 allocs/op
BenchmarkLogWithFieldsMerge-10            853112              1362 ns/op            1435 B/op         10 allocs/op
BenchmarkLogParallel-10                  1000000              1153 ns/op            1276 B/op         10 allocs/op
BenchmarkLogWithCallerInfo-10            2143800               557.8 ns/op           857 B/op          8 allocs/op
BenchmarkLogNoOutput-10                  4499454               266.4 ns/op           296 B/op          3 allocs/op
```