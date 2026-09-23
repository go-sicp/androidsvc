//go:build !android

package jni

// Non-Android stub: these definitions exist purely so the package compiles
// (and `go vet` / IDE tooling work) on developer machines. All operations
// return ErrNotReady; nothing is wired to a real JVM.

func IsReady() bool { return false }

func SetServiceCallbacks(create func(int64), start func(string) int32, destroy func()) {}

func Do(fn func(Env) error) error { return ErrNotReady }

func AdoptGlobal(handle int64) GlobalRef { return nil }
