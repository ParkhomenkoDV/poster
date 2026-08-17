# Progress

Пакет `progress` предоставляет прогресс-бар с периодическим обновлением для Go-приложений.

## Install

```bash
go get github.com/ParkhomenkoDV/progress
```

## Usage

### Fields

| Поле         | Тип             | Описание                                                |
|--------------|-----------------|---------------------------------------------------------|
| `Interval`   | `time.Duration` | Частота обновления.                                     |
| `Description`| `string`        | Текст, выводимый перед прогресс-баром.                  |
| `Length`     | `uint8`         | Длина шкалы в символах (0 – не отображать).             |
| `Total`      | `uint64`        | Общее количество единиц работы (0 – неизвестно).        |
| `ShowETA`    | `bool`          | Показывать оценочное время до завершения.               |
| `ShowSpeed`  | `bool`          | Показывать скорость обработки (шт/с).                   |
| `Leave`      | `bool`          | Оставить прогресс полсе завершения.                     |

### Auto
```go
package main

import (
    "context"
    "sync/atomic"
    "time"
    
    "github.com/ParkhomenkoDV/progress"
)

func main() {
    var (
        bar = progress.New(1*time.Second, "Loading", 50, 100, true, true, true)
    ) 
    defer bar.Start(context.Background())()

    // Симуляция работы
    go func() {
        for i := 0; i < 1000; i++ {
            bar.Add(1)
            if err := doWork(); err != nil {
                bar.AddError(1)
            }
        }
    }()
}
```

### Manual
```go
package main

import (
    "context"
    "sync/atomic"
    "time"
    
    "github.com/ParkhomenkoDV/progress"
)

func main() {
    var (
        wgBar         sync.WaitGroup // ждун
        bar = progress.New(
            500*time.Millisecond, // интервал обновления
            "Loading",            // описание
            20,                   // длина шкалы
            100,                  // всего элементов
            true,                 // показывать ETA
            true,                 // показывать скорость
            true,                 // оставить прогресс после завершения
        )
        ctxBar, cancelBar = context.WithCancel(context.Background())
    ) 
    wgBar.Add(1) 
    go func() {
        defer wgBar.Done()
        bar.Show(ctxBar)
    }()

    go func() {
        // Симуляция работы
        for i := 0; i < 100; i++ {
            bar.Add(1)
            time.Sleep(100 * time.Millisecond)
        }
    }()

    cancelBar()
    wgBar.Wait()
}
```

## Result

```
Loading 100% |-------   | 69/100 ❌ 0 ⏰ 0h3m15s ⚡️ 100.0 it/s 
```

## Benchmarks
```
goos: darwin
goarch: arm64
pkg: github.com/ParkhomenkoDV/progress
cpu: Apple M4
BenchmarkAdd-10                 40353370                33.38 ns/op            0 B/op          0 allocs/op
BenchmarkAddError-10            34805770                35.08 ns/op            0 B/op          0 allocs/op
BenchmarkGetLoad-10             42778970                26.70 ns/op           64 B/op          1 allocs/op
BenchmarkFormatDuration-10       9809126               122.1 ns/op            48 B/op          6 allocs/op
```

## License

MIT