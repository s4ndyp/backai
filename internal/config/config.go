package config

import (
	"os"
	"strconv"
	"time"
)

type Server struct {
	HTTPAddr         string
	DatabaseURL      string
	SessionSecret    string
	AgentRegisterKey string
	MigrateOnStart   bool
	JobPollInterval  time.Duration
}

func LoadServer() Server {
	return Server{
		HTTPAddr:         env("BACKAI_HTTP_ADDR", ":8080"),
		DatabaseURL:      env("BACKAI_DATABASE_URL", "postgres://backai:backai@localhost:5432/backai?sslmode=disable"),
		SessionSecret:    env("BACKAI_SESSION_SECRET", "change-me-in-production"),
		AgentRegisterKey: env("BACKAI_AGENT_REGISTER_KEY", "change-agent-register-key"),
		MigrateOnStart:   envBool("BACKAI_MIGRATE_ON_START", true),
		JobPollInterval:  envDuration("BACKAI_JOB_POLL_INTERVAL", 30*time.Second),
	}
}

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
		ServerURL:   env("BACKAI_SERVER_URL", "http://localhost:8080"),
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

func envBool(k string, def bool) bool {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
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
