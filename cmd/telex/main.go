package main

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	Execute(buildInfo{Version: version, Commit: commit, Date: date})
}
