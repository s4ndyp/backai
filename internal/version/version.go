package version

// Set at link time via -ldflags.
var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

func AgentUserAgent() string {
	return "backai-agent/" + Version
}

func ServerUserAgent() string {
	return "backai-server/" + Version
}
