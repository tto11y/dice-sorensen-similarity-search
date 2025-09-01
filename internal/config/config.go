package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"go.uber.org/zap/zapcore"
	"net/url"
	"os"
	"time"
)

type JsonUrl struct {
	*url.URL
}

func (j *JsonUrl) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	configUrl, err := url.Parse(s)
	j.URL = configUrl
	return err
}

func (j *JsonUrl) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.URL.String())
}

type JsonDuration struct {
	time.Duration
}

func (j *JsonDuration) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	var duration time.Duration
	duration, err = time.ParseDuration(s)
	if err != nil {
		return err
	}
	j.Duration = duration
	return err
}

type Configuration struct {
	Logging struct {
		MaxSize         int
		MaxBackups      int
		MaxAge          int
		Level           zapcore.Level
		ConsoleLogLevel zapcore.Level
		File            string
		HttpAccessFile  string
		DbLogFile       string
		LogAlerts       bool
	}
	ListeningPort    string
	ListeningAddress string
	Database         struct {
		Host            string
		Port            uint
		Username        string
		Password        string
		DatabaseName    string
		MaxIdleConns    int
		MaxOpenConns    int
		ConnMaxLifetime *JsonDuration
	}
	BitBucket struct {
		Url         *JsonUrl
		User        string
		Password    string
		AccessToken string
		ProjectName string
		Repository  string
	}
}

var config *Configuration

func InitConfig() *Configuration {
	configFile := flag.String("config", "config.json", "Path to config file (json)")
	flag.Parse()
	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "\nUsage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		_, _ = fmt.Fprint(os.Stderr, "\n")
	}

	file, _ := os.Open(*configFile)
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		flag.Usage()
		panic("Error parsing config file: " + err.Error())
	}

	//defaults
	if config.Logging.MaxSize <= 0 {
		config.Logging.MaxSize = 500
	}
	if config.Logging.MaxBackups <= 0 {
		config.Logging.MaxBackups = 3
	}
	if config.Logging.MaxAge <= 0 {
		config.Logging.MaxAge = 28
	}

	return config
}

func Config() *Configuration {
	return config
}

func Port() string {
	return config.ListeningPort
}

func Address() string {
	return config.ListeningAddress
}

func DbHost() string {
	return config.Database.Host
}

func DbName() string {
	return config.Database.DatabaseName
}

func DbUser() string {
	return config.Database.Username
}

func DbPassword() string {
	return config.Database.Password
}
