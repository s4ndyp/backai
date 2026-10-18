package config

import (
	"os"
	"time"
)

type Agent struct {
	ServerURL   string
	RegisterKey string
	Name        string
	Token       string
	PollEvery   time.Duration
	ResticPath  string
}

func LoadAgent() Agent {
	return Agent{
		ServerURL:   env("BACKAI_SERVER_URL", "http://localhost:8090"),
		RegisterKey: env("BACKAI_AGENT_REGISTER_KEY", ""),
		Name:        env("BACKAI_AGENT_NAME", ""),
		Token:       env("BACKAI_AGENT_TOKEN", ""),
		PollEvery:   envDuration("BACKAI_AGENT_POLL_INTERVAL", 15*time.Second),
		ResticPath:  env("BACKAI_RESTIC_PATH", "restic"),
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envDuration(k string, def time.Duration) time.Duration {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
