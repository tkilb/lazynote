// Package version holds the para build version. Version defaults to
// "dev" for local `go build`/`go run` and is overridden at release-build
// time via `-ldflags "-X github.com/tkilb/para/internal/version.Version=X.Y.Z"`
// (see the Makefile's `build` target, which reads the VERSION file).
package version

// Version is the para release version, e.g. "0.0.1".
var Version = "dev"
