# Poster

Массовая отправка HTTP запросов и получение ответов с последующим сохранением

![](./assets/images/poster.jpg)

## Requirements

- Go 1.18 или выше
- Сервер, принимающий POST-запросы с `Content-Type`: `application/json`

## Usage

1. Положить в корень проекта `.` директорию `requests` с запросами
2. Поднять сервер по адресу `URL`
 
```bash
go run poster.go [--url <URL>] [--requests <имяДиректорииЗапросов>] [--responses <имяДиректорииОтветов>] [--timeout N] [--workers N]
```

Флаг | Описание | По умолчанию
---|---|---
URL | URL сервера для отправки запросов | http://localhost:8080/execute
requests | Директория с JSON-файлами запросов | requests
responses | Директория для сохранения ответов | responses
timeout | Таймаут HTTP-запросов (секунды) | 3
workers | Количество параллельных воркеров | количетсво ядер - 1

3. Результат прогона находится в директории `responses`

## Limitations

- Поддерживаются только POST-запросы
- Все запросы отправляются на один `URL`
- Максимальное количество одновременных запросов ограничено параметром `workers`

## Build

```bash
go build -o poster poster.go
./poster
```

## Structure
```
poster/
├── poster.go             # Основной файл программы
├── requests/             # Директория с запросами
│   ├── request1.json
│   ├── request2.json
│   └── ...
├── responses/            # Директория с ответами (создается автоматически)
│   ├── request1.json
│   ├── request2.json
│   └── ...
├── internal/
│   ├── config/
│   │   └── config.go     # Конфигурация программы
│   │   └── flags.go      # Флаги программы
│   └── ...
├── go.mod                # Модуль Go
└── README.md             # Документация
```

