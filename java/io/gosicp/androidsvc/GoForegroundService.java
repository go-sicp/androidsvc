package io.gosicp.androidsvc;

import android.app.Notification;
import android.app.Service;
import android.content.Intent;
import android.os.Build;
import android.os.IBinder;

/**
 * GoForegroundService is the Android-side foreground service whose lifecycle
 * is mirrored into Go. The actual logic lives in the Go package
 * github.com/go-sicp/androidsvc, dispatched here via JNI native methods
 * resolved against libgojni.so.
 *
 * Declare it in your AndroidManifest.xml:
 * <pre>
 * &lt;service
 *     android:name="io.gosicp.androidsvc.GoForegroundService"
 *     android:exported="false"
 *     android:foregroundServiceType="dataSync" /&gt;
 * </pre>
 */
public class GoForegroundService extends Service {

    static {
        System.loadLibrary("gojni");
    }

    @Override
    public void onCreate() {
        super.onCreate();
        // Make sure the JNI bridge is initialized even if the Service is
        // the first Go-touching component to run (boot completed, alarm).
        GoServiceBridge.init(getApplicationContext());
        nativeOnCreate();
    }

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        String action = (intent != null && intent.getAction() != null)
                ? intent.getAction() : "";
        return nativeOnStartCommand(action);
    }

    @Override
    public void onDestroy() {
        try {
            nativeOnDestroy();
        } finally {
            super.onDestroy();
        }
    }

    @Override
    public IBinder onBind(Intent intent) {
        // v0.1: started service only. v0.2 will return a Binder backed by
        // androidsvc/control for UI ↔ Service RPC.
        return null;
    }

    /**
     * Called from Go to enter foreground state with a built Notification.
     *
     * @param fgType ServiceInfo.FOREGROUND_SERVICE_TYPE_* bitmask, or 0 to
     *               fall back to the no-type variant (Android 13 still
     *               accepts that; Android 14 will not).
     */
    public void doStartForeground(int id, Notification n, int fgType) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q && fgType != 0) {
            startForeground(id, n, fgType);
        } else {
            startForeground(id, n);
        }
    }

    private native void nativeOnCreate();
    private native int  nativeOnStartCommand(String action);
    private native void nativeOnDestroy();
}
