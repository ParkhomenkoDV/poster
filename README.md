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
go run poster.go [-url <URL>] [-requests <имяДиректорииЗапросов>] [-responses <имяДиректорииОтветов>] [-indent] [-timeout N] [-workers N] [-log S]
```

Флаг | Описание | По умолчанию
---|---|---
`url`       | URL сервера для отправки запросов  | http://localhost:8080/execute
`requests`  | Директория с JSON-файлами запросов | requests
`responses` | Директория для сохранения ответов  | responses
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
├── poster.go             # Основной файл программы
├── requests/             # Директория с запросами
│   ├── 1.json
│   ├── 2.json
│   └── ...
├── responses/            # Директория с ответами (создается автоматически)
│   ├── 1.json
│   ├── 2.json
│   └── ...
├── internal/
│   ├── config/
│   │   └── config.go     # Конфигурация программы
│   │   └── flags.go      # Флаги программы
│   └── ...
├── go.mod                # Модуль Go
└── README.md             # Документация
```

