//go:build tools

// This file anchors build-time dependencies that are not imported by any
// runtime Go file. Without it, `go mod tidy` drops these from go.mod.
package androidsvc

import _ "golang.org/x/mobile/bind"
