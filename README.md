# Poster

Отправка request и получение response

![](./assets/images/poster.jpg)

## Usage

1. Поднять сервер по адресу `URL`
2. Положить в корень проекта `.` директорию `requests` с запросами 

```bash
go run poster.go [--url <URL>] [--requests <requests>] [--responses <responses>] [--timeout N]
```

3. Результат прогона находится в директории `responses`

## Structure
```
poster/
|-- poster.go
|-- requests/
|   |-- request1.json
|   |-- request2.json
|   └-- ...
|-- responses/
|   |-- response1.json
|   |-- response2.json
|   └-- ...
|-- internal/
|   |-- config.go
|   |-- flags.go
```

