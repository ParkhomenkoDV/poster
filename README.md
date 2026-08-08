# Poster

Массовая отправка HTTP запросов и получение ответов с последующим сохранением

![](./assets/images/poster.jpg)

## Requirements

- Go 1.25 или выше
- Сервер, принимающий POST-запросы с `Content-Type`: `application/json`

## Usage

1. Поднять сервер по адресу `URL`
2. Запустить Poster командой:
 
```bash
go run poster.go [-url <URL>] [-requests <dir>] [-indent] [-timeout N] [-workers N] [-log S]
```

Флаг        | Описание                           | По умолчанию
------------|------------------------------------|------------------------------
`url`       | URL сервера для отправки запросов  | http://localhost:8080/execute
`requests`  | Директория с JSON-файлами запросов | ./requests
`indent`    | Форматирование ответа              | false
`timeout`   | Таймаут HTTP-запросов (секунды)    | 10
`workers`   | Количество параллельных воркеров   | количетсво ядер
`log`       | Уровень логирования ('', 'stdout', 'debug', 'info', 'warn', 'error', 'fatal') | ''

3. Результат прогона находится в директории `responses`

## Build

```bash
go build -o poster poster.go
./poster
```

## Structure
```
poster/
├── internal/
│   ├── config/
│   │   └── config.go     # Конфигурация программы
│   │   └── flags.go      # Флаги программы
│   ├── logger/
│   │   └── logger.go     # Логгер
│   ├── progress/
│   │   └── progress.go   # Прогресс бар
├── requests/             # Директория с запросами
│   ├── 1.json
│   ├── 2.json
│   └── ...
├── responses/            # Директория с ответами (создается автоматически)
│   ├── 1.json
│   ├── 2.json
│   └── ...
├── vendor/               # Внешние зависисмости
│   └── ...
├── go.mod                # Модуль Go
├── Makefile              # Команды
├── poster.go             # Основной файл программы
└── README.md             # Документация
```

## Architecture
```
├── requests/
│   ├── 1.json
│   ├── 2.json
│   └── ...

       |
       v

     worker
     ------
    | read |
    | post |
    | save |
     ------

       |
       v

├── responses/
│   ├── 1.json
│   ├── 2.json
│   └── ...
```

## Benchmarks
```
goos: darwin
goarch: arm64
pkg: poster
cpu: Apple M4
BenchmarkReadFile/single/1KB-10                   135253              7758 ns/op             408 B/op          4 allocs/op
BenchmarkReadFile/single/100KB-10                 116854             10296 ns/op             409 B/op          4 allocs/op
BenchmarkReadFile/single/1MB-10                    41547             29757 ns/op             458 B/op          4 allocs/op
BenchmarkReadFile/single/10MB-10                    2811            409491 ns/op            7868 B/op          4 allocs/op
BenchmarkReadFile/buffer-sizes/cap-512-10                  41040             29649 ns/op             459 B/op          4 allocs/op
BenchmarkReadFile/buffer-sizes/cap-1024-10                 40477             29647 ns/op             459 B/op          4 allocs/op
BenchmarkReadFile/buffer-sizes/cap-4096-10                 38209             29933 ns/op             462 B/op          4 allocs/op
BenchmarkReadFile/buffer-sizes/cap-16384-10                40681             29696 ns/op             459 B/op          4 allocs/op
BenchmarkReadFile/buffer-sizes/cap-65536-10                40374             29751 ns/op             459 B/op          4 allocs/op
BenchmarkReadFile/buffer-sizes/cap-1048576-10              40452             29609 ns/op             408 B/op          4 allocs/op
BenchmarkReadFile/parallel-10                              52250             21607 ns/op             809 B/op          4 allocs/op
BenchmarkSendRequest/small-10                              39822             29714 ns/op            7962 B/op         85 allocs/op
BenchmarkSendRequest/medium-10                             40083             29648 ns/op           10361 B/op         88 allocs/op
BenchmarkSendRequest/large-10                              33394             35902 ns/op           62413 B/op         98 allocs/op
Benchmark_saveResponse/indent-10                           42642             32570 ns/op            1768 B/op          7 allocs/op
Benchmark_saveResponse/no-indent-10                        43988             29492 ns/op             328 B/op          4 allocs/op
```