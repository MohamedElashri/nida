package buildinfo

import "fmt"

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
	BuiltBy = "unknown"
)

func Summary() string {
	return fmt.Sprintf("nida version %s (commit=%s date=%s builtBy=%s)", Version, Commit, Date, BuiltBy)
}
