package config

var Service = "global-probe"
var Version = "v1.0.0"
var GitCommit string
var OSBuildName string
var BuildDate string

type Timeout struct {
	Connect     int `yaml:"connect" validate:"required,min=1,max=60"`
	Transaction int `yaml:"transaction" validate:"required,min=1,max=300"`
}

type Pool struct {
	Producer int `yaml:"producer" validate:"required,min=1,max=500"`
	Consumer int `yaml:"consumer" validate:"required,min=1,max=50000"`
}

type MainConfig struct {
	Log struct {
		Level  string `yaml:"level" validate:"oneof=trace debug info warn error fatal panic"`
		Format string `yaml:"format" validate:"oneof=text json"`
	} `yaml:"log"`
	Probe struct {
		NormalTimeout float64 `yaml:"normalTimeout" validate:"required,min=1,max=60"`
	} `yaml:"probe"`
	APM struct {
		Enabled bool     `yaml:"enabled"`
		Host    string   `yaml:"host"`
		Port    int      `yaml:"port" validate:"required,min=1,max=65535"`
		Rate    *float64 `yaml:"rate" validate:"omitempty,min=0.1,max=1"`
	} `yaml:"apm"`
	App struct {
		Name    string `yaml:"name" validate:"required"`
		Version string `yaml:"version" validate:"required"`
		Env     string `yaml:"env" validate:"required"`
		Tribe   string `yaml:"tribe" validate:"required"`
	}
}

type Module struct {
	Name   string            `yaml:"name"`
	Config map[string]string `yaml:"config"`
}

type WebsiteConfig struct {
	Method        string            `yaml:"method"`
	Authorization *Authorization    `yaml:"authorization"`
	Headers       map[string]string `yaml:"headers"`
	Body          string            `yaml:"body"`
	Query         map[string]string `yaml:"query"`
}

type WorkerProbe struct {
	Tribe        string         `yaml:"tribe"`
	Operation    string         `yaml:"operation"`
	Order        int            `yaml:"order"`
	Ip           string         `yaml:"ip"`
	Dependencies []string       `yaml:"dependencies"`
	Interval     int            `yaml:"interval"`
	Modules      []*Module      `yaml:"modules"`
	ProbeConfig  *WebsiteConfig `yaml:"probe_config"`
}

type ProbesConfig struct {
	Probes []WorkerProbe `yaml:"probes"`
}

type Authorization struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}
