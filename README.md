# androidsvc

Drive an Android foreground service from Go, alongside an AAR produced by
`gomobile bind`. Targets Android 13 (API 33), supports API 26+.

Status: **v0.1 — functional skeleton, not yet validated on device**. The
package compiles in stub mode on darwin/linux; real-device validation
requires building an AAR with `gomobile bind -target=android` from a
consumer project and embedding it in an Android Studio module.

## Why

`golang.org/x/mobile` is in maintenance mode and its `mobileinit` package
is import-restricted (`internal/`). This library provides the minimum
plumbing needed to drive a Service from Go (capture JavaVM, attach thread,
JNI helpers, lifecycle) without patching x/mobile.

## Architecture

```
github.com/go-sicp/androidsvc                ← Service interface, lifecycle
├── notif/                                   ← NotificationChannel, Notification
├── log/                                     ← ring buffer + slog.Handler
├── internal/jni/                            ← cgo bridge, JNI helpers
└── java/io/gosicp/androidsvc/               ← Java companion classes
    ├── GoServiceBridge.java                 ← JNI init + start/stop helpers
    └── GoForegroundService.java             ← Android Service + JNI callbacks
```

Nothing is imported from `golang.org/x/mobile/internal/*`. The JavaVM is
captured via `GoServiceBridge.nativeInit(Context)`, which resolves to the
C symbol `Java_io_gosicp_androidsvc_GoServiceBridge_nativeInit` defined in
the cgo of `internal/jni/jni_android.go`. At `gomobile bind` time, that C
code is compiled into the same `libgojni.so` as the generated Go bindings.

## Setup in the host Android project

### 1. Build the AAR

The Go consumer package (which must `import "github.com/go-sicp/androidsvc"`)
is fed to `gomobile bind`:

```bash
gomobile bind -target=android \
              -androidapi=26 \
              -o myapp.aar \
              github.com/your-org/your-go-package
```

`-androidapi=26` is the floor (NotificationChannel). The `targetSdk` is
configured in the Android module, not here.

### 2. Wire it into the app module

- Drop the generated AAR into the Android module's `libs/`.
- Copy `java/io/gosicp/androidsvc/*.java` into `app/src/main/java/`
  (or package them in a separate Android library — anything that places
  them on the app classpath is fine).
- Merge `manifest/manifest_snippet.xml` into your `AndroidManifest.xml`.

### 3. Runtime permissions (Android 13)

`POST_NOTIFICATIONS` must be requested at runtime from an Activity:

```kotlin
if (ContextCompat.checkSelfPermission(this, POST_NOTIFICATIONS) != GRANTED) {
    requestPermissions(arrayOf(POST_NOTIFICATIONS), 0)
}
```

Do not call `GoServiceBridge.start(ctx)` before the permission is granted:
otherwise Android 13 silently drops the notification and kills the service
after roughly five seconds.

### 4. Start

In your `Application.onCreate()`:

```kotlin
override fun onCreate() {
    super.onCreate()
    GoServiceBridge.init(this)
    Myapp.initService()           // exported by gomobile bind
}
```

In your Go consumer package:

```go
package myapp

import (
    "log/slog"

    "github.com/go-sicp/androidsvc"
    asvclog "github.com/go-sicp/androidsvc/log"
    "github.com/go-sicp/androidsvc/notif"
)

var Service androidsvc.Service

func InitService() {
    ch := notif.NewChannel(
        notif.WithChannelID("main"),
        notif.WithChannelName("Main service"),
        notif.WithImportance(notif.ImportanceLow),
    )
    Service = androidsvc.New(
        androidsvc.WithChannel(ch),
        androidsvc.WithLogCapacity(2000),
        androidsvc.WithNotificationContent("My service", "Running", 0),
        androidsvc.WithForegroundServiceType(1), // FOREGROUND_SERVICE_TYPE_DATA_SYNC
    )
    androidsvc.Register(Service)

    // Plug slog into the in-memory ring buffer so the UI can display logs.
    slog.SetDefault(slog.New(asvclog.NewHandler(Service.Logs(), slog.LevelInfo)))
}

func Start() error { return Service.Start(nil) }
func Stop() error  { return Service.Stop() }
```

From the UI (Activity / Compose):

```kotlin
fun onStartClicked() = Myapp.start()
fun onStopClicked()  = Myapp.stop()
```

## Streaming logs to the UI (v0.1)

`Service.Logs()` returns a subscribable `log.Buffer`:

```go
// Exported for gomobile bind.
func StreamLogs(cb LogCallback) func() {
    ch, cancel := myapp.Service.Logs().Subscribe(64)
    go func() {
        for e := range ch {
            cb.OnEntry(e.Time().UnixMilli(), int32(e.Level()), e.Message())
        }
    }()
    return cancel
}

// LogCallback is a Go interface — gomobile bind exposes it as a Java
// interface that the Kotlin side implements.
type LogCallback interface {
    OnEntry(unixMillis int64, level int32, msg string)
}
```

The Compose UI implements `LogCallback` and pushes entries into a
`SnapshotStateList`.

## Editing config (v0.1)

v0.1 does not yet ship a typed config store. Recommended pattern until
`androidsvc/config` (v0.2):

- expose `func GetConfig() string` / `func SetConfig(json string) error`
- persist to a file under `Context.getFilesDir()`
- the UI reads/writes via gomobile bind, then triggers a Go-side reload.

## Known v0.1 limits

- **No Binder IPC between UI and service**: for now everything goes
  through direct Go calls from the UI. This works as long as both UI and
  service live in the same process (the default). `Binder` support lands
  in v0.2.
- **No `BroadcastReceiver`**: cannot react to `BOOT_COMPLETED` or
  connectivity changes from Go. Workaround: a Java receiver that calls
  `GoServiceBridge.start(ctx)`.
- **No `WorkManager` / `AlarmManager`** wrapper.
- **Runtime permissions**: must be requested from the Java/Kotlin side
  (Activity required).
- **No handling for `ForegroundServiceStartNotAllowedException`**
  (Android 12+): start the service only from a foreground Activity.
- **`onBind` returns null**: no bound service in v0.1.

## Roadmap

| Version | Scope |
|---------|-------|
| v0.1    | JNI bridge, NotificationChannel, foreground start/stop, log buffer + slog |
| v0.2    | `control` (Binder JSON-RPC), `config` (SharedPreferences/file), perm helpers via Activity |
| v0.3    | `BroadcastReceiver` bridge, `WorkManager`, BootCompleted helper |
| v1.0    | Stable API, end-to-end Compose+Go example, emulator integration tests |

## Local validation

Without an NDK installed, only the stub build can be exercised:

```bash
task check      # fmt:check + vet + test + staticcheck
```

With the Android command-line tools and NDK installed, the cgo + JNI side
can be cross-compiled in place to flag preamble / linker issues without
having to bind a full AAR:

```bash
export ANDROID_HOME=/opt/homebrew/share/android-commandlinetools
export ANDROID_NDK_HOME=$ANDROID_HOME/ndk/27.0.12077973
export JAVA_HOME=/opt/homebrew/opt/openjdk@21/libexec/openjdk.jdk/Contents/Home
task android:smoke   # GOOS=android GOARCH=arm64 go build ./...
task java:check      # javac the companion classes against android.jar
```

A real Android binding runs `gomobile bind` from a consumer project that
imports this package — see `tableaux-eink-go/einkapp` for a working
example.

## License

MIT (to be confirmed).
