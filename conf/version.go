package conf

var (
	// Executable is overridden by Makefile
	Executable = "penny"

	// GitVersion is overridden at build time via ldflags
	GitVersion = "dev"
)
