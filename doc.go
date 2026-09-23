// Package androidsvc lets Go code drive an Android foreground service from
// inside a gomobile-bind AAR. It targets Android 13 (API 33) as the primary
// platform but works back to API 26 (Oreo, NotificationChannel).
//
// It is meant to sit alongside golang.org/x/mobile (gomobile bind) without
// patching it: the cgo bridge captures the JavaVM via a Java callback rather
// than relying on the unexported mobileinit package.
//
// See README.md for build, manifest and Java glue setup.
package androidsvc
