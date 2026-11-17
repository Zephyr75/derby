//go:build !darwin

package main

// useLoginShellPath is only needed on macOS (see shellpath_darwin.go).
func useLoginShellPath() {}
