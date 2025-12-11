package version

import "runtime"

var build = "Unknown"
var gitRev = "Unknown"
var gitBranch = "Unknown"
var gitTag = "Unknown"

type BuildInfo struct {
	BuildTime string
	GitBranch string
	GitRev    string
	GoVersion string
	GitTag    string
}

func GetBuildInfo() BuildInfo {
	return BuildInfo{
		BuildTime: build,
		GitBranch: gitBranch,
		GitRev:    gitRev,
		GoVersion: runtime.Version(),
		GitTag:    gitTag,
	}
}
