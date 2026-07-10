//go:build !windows

package cmds

import "errors"

var errWindowsOnly = errors.New("service mode is only supported on Windows")

func ServiceRun() error                 { return errWindowsOnly }
func ServiceInstall() error             { return errWindowsOnly }
func ServiceUninstall(purge bool) error { return errWindowsOnly }
func ServiceStart() error               { return errWindowsOnly }
func ServiceStop() error                { return errWindowsOnly }
func ServiceStatus() error              { return errWindowsOnly }
