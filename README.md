# Poster

Отправка request и получение response

![](./assets/images/poster.jpg)

## Usage

```bash
go run poster.go [--url <URL>] [--requests <имяДиректории>] [--responses <имяДиректории>] [--timeout N]
```

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

