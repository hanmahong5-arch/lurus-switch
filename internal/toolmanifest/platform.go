package toolmanifest

import "runtime"

// CurrentPlatform returns the current OS/arch string used as manifest platform key.
// Windows ARM64 is mapped to "windows/amd64" because lurus-switch ships only x64
// binaries for Windows (the x64 compatibility layer handles them transparently).
func CurrentPlatform() string {
	goos := runtime.GOOS
	arch := runtime.GOARCH
	if goos == "windows" && arch == "arm64" {
		arch = "amd64"
	}
	return goos + "/" + arch
}

// IsSupportedPlatform returns (true, "") for supported platforms.
// Returns (false, reason) for unsupported ones so the caller can surface a friendly
// error rather than a confusing install failure.
func IsSupportedPlatform() (bool, string) {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "windows/386", "linux/386":
		return false, "lurus-switch requires a 64-bit operating system (current: " + runtime.GOOS + "/386)"
	}
	return true, ""
}
