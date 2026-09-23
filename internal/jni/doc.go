// Package jni is a thin, self-contained bridge between Go and the JVM on
// Android. It captures the JavaVM and the application Context via a Java
// callback (GoServiceBridge.nativeInit) and exposes a Do(fn) entry point
// that runs fn with a freshly attached JNIEnv.
//
// The package does NOT depend on golang.org/x/mobile/internal/mobileinit
// (which is import-restricted). It mirrors the JNI helper pattern used by
// gioui.org/app, adapted for use from a gomobile-bind AAR.
//
// All Class, Object and MethodID handles returned from Env methods are only
// valid for the duration of the enclosing Do call. Use NewGlobalRef to
// retain a reference across calls.
package jni
