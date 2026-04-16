package sysdetect

import (
	"os"
	"strings"
)

// DeployType describes how mediamtx is deployed on the host.
type DeployType string

const (
	DeployDocker     DeployType = "docker"
	DeployPodman     DeployType = "podman"
	DeployCompose    DeployType = "compose"
	DeployQuadlets   DeployType = "quadlets"
	DeploySystemd    DeployType = "systemd"
	DeployUnknown    DeployType = "unknown"
)

// Detect returns the best guess at how mediamtx is deployed.
// The MEDIAMTX_UI_DEPLOY_TYPE env var overrides auto-detection.
func Detect() DeployType {
	if override := os.Getenv("MEDIAMTX_UI_DEPLOY_TYPE"); override != "" {
		return DeployType(override)
	}

	// Check for quadlet unit files (Podman quadlets live in these dirs)
	quadletDirs := []string{
		"/etc/containers/systemd",
		"/usr/share/containers/systemd",
	}
	for _, dir := range quadletDirs {
		if hasContainerFiles(dir) {
			return DeployQuadlets
		}
	}

	// Check cgroup for container runtime
	cgroup, _ := os.ReadFile("/proc/1/cgroup")
	cgroupStr := string(cgroup)
	if strings.Contains(cgroupStr, "docker") {
		return DeployDocker
	}
	if strings.Contains(cgroupStr, "libpod") || strings.Contains(cgroupStr, "podman") {
		return DeployPodman
	}

	// Check for docker-compose / compose file in common locations
	composePaths := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
		"/opt/mediamtx/docker-compose.yml",
		"/opt/mediamtx/compose.yml",
	}
	for _, p := range composePaths {
		if fileExists(p) {
			return DeployCompose
		}
	}

	// Check if mediamtx is managed by systemd
	if fileExists("/etc/systemd/system/mediamtx.service") ||
		fileExists("/lib/systemd/system/mediamtx.service") {
		return DeploySystemd
	}

	return DeployUnknown
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func hasContainerFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".container") {
			return true
		}
	}
	return false
}
