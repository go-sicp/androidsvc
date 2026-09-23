package io.gosicp.androidsvc;

import android.content.Context;
import android.content.Intent;
import android.os.Build;

/**
 * GoServiceBridge is the entry point between the Android side and the Go
 * runtime embedded in libgojni.so (produced by `gomobile bind`).
 *
 * Call {@link #init(Context)} once during app startup (e.g. from your
 * Application.onCreate) <em>before</em> any other interaction with the Go
 * service: it captures the JavaVM and the application Context on the Go side.
 *
 * It is also called automatically from {@link GoForegroundService#onCreate()}
 * so the bridge is initialized even when the Service starts before any
 * Activity (boot-completed receivers, alarms, …).
 */
public final class GoServiceBridge {

    static {
        // libgojni.so is the artifact produced by `gomobile bind -target=android`.
        // It contains both the Go runtime and the C symbols defined in
        // androidsvc/internal/jni/jni_android.go.
        System.loadLibrary("gojni");
    }

    private static boolean initialized = false;

    private GoServiceBridge() {}

    /** Idempotent JNI bridge initialization. */
    public static synchronized void init(Context ctx) {
        if (initialized) return;
        nativeInit(ctx.getApplicationContext());
        initialized = true;
    }

    /** Starts the Go-controlled foreground service. */
    public static void start(Context ctx) {
        Intent intent = new Intent(ctx, GoForegroundService.class);
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            ctx.startForegroundService(intent);
        } else {
            ctx.startService(intent);
        }
    }

    /** Stops the Go-controlled foreground service. */
    public static void stop(Context ctx) {
        Intent intent = new Intent(ctx, GoForegroundService.class);
        ctx.stopService(intent);
    }

    /** Implemented in libgojni.so (cgo). Captures JavaVM, Context, ClassLoader. */
    private static native void nativeInit(Context appCtx);
}
