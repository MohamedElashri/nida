package buildinfo

import "fmt"

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
	BuiltBy = "unknown"
)

func Summary() string {
	if BuiltBy == "goreleaser" && Version != "dev" {
		return fmt.Sprintf("nida version %s", Version)
	}
	return fmt.Sprintf("nida version %s (commit=%s date=%s builtBy=%s)", Version, Commit, Date, BuiltBy)
}
