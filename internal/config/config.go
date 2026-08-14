package config

type Config struct {
	URL     string `doc:"Адрес сервера"`
	Req     string `doc:"Директория с запросами json"`
	Indent  bool   `doc:"Форматирование ответа"`
	Timeout int    `doc:"Max время для ответа"`
	Workers int    `doc:"Количество параллельных работников"`
	Log     string `doc:"Уровень логирования ('', 'stdout', 'debug', 'info', 'warn', 'error')"`
}

func New() (*Config, error) {
	flags, err := parse()
	if err != nil {
		return &Config{}, err
	}

	return &Config{
		URL:     flags.URL,
		Req:     flags.Req,
		Indent:  flags.Indent,
		Timeout: flags.Timeout,
		Workers: flags.Workers,
		Log:     flags.Log,
	}, nil
}
