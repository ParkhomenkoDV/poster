package config

type Config struct {
	URL          string `doc:"Адрес сервера"`
	RequestsDir  string `doc:"Директория с запросами json"`
	ResponsesDir string `doc:"Директория с ответами json"`
	Timeout      int    `doc:"Max время для ответа"`
}

func New() (*Config, error) {
	flags, err := parse()
	if err != nil {
		return &Config{}, err
	}

	return &Config{
		URL:          flags.URL,
		RequestsDir:  flags.RequestsDir,
		ResponsesDir: flags.ResponsesDir,
		Timeout:      flags.Timeout,
	}, nil
}
