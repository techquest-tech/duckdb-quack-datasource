//go:build mage

package main

import (
	"os"
	"os/exec"
	"runtime"

	"github.com/magefile/mage/mg"
)

const pluginID = "panarm-duckdb-datasource"

func buildFrontend() error {
	cmd := exec.Command("npm", "run", "build")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func buildBackendGo(goos, goarch string) error {
	cmd := exec.Command("go", "build", "-ldflags", "-s -w",
		"-o", "dist/gpx_"+pluginID+"_"+goos+"_"+goarch, "./cmd")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+goos, "GOARCH="+goarch, "GOWORK=off")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

// Build assembles the plugin: frontend (webpack -> dist/module.js + .map) then
// backend (Go -> gpx_<plugin>_<os>_<arch>).
func Build() error {
	mg.Deps(BuildFrontend, BuildBackend)
	mg.Deps(Dist)
	return nil
}

func BuildFrontend() error { return buildFrontend() }

func BuildBackend() error {
	if err := buildBackendGo("linux", "amd64"); err != nil {
		return err
	}
	if runtime.GOOS == "darwin" {
		_ = buildBackendGo("darwin", "arm64")
	}
	return nil
}

// Dist packages the plugin into dist/<pluginID> for release/signature.
func Dist() error {
	mg.Deps(BuildFrontend, BuildBackend)
	return nil
}

func main() {
	os.Exit(mg.Main())
}
