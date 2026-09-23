// Native JNI trampolines for the Java companion classes (GoServiceBridge,
// GoForegroundService). They live in this .c file rather than in the cgo
// preamble of jni_android.go so they are compiled exactly once: cgo would
// otherwise emit them in both _cgo_export.c (because jni_android.go has
// //export directives) and the regular cgo .c, producing duplicate symbols.

#include <jni.h>
#include <stdlib.h>
#include <string.h>
#include <android/log.h>
#include "_cgo_export.h"

#define ASVC_TAG "androidsvc"
#define ASVC_LOGI(...) __android_log_print(ANDROID_LOG_INFO, ASVC_TAG, __VA_ARGS__)

JavaVM   *g_vm           = NULL;
jobject   g_app_ctx      = NULL;
jobject   g_class_loader = NULL;
jmethodID g_load_class   = NULL;

JNIEXPORT void JNICALL
Java_io_gosicp_androidsvc_GoServiceBridge_nativeInit(JNIEnv *env, jclass cls, jobject ctx) {
    (void)cls;
    if (g_vm == NULL) {
        (*env)->GetJavaVM(env, &g_vm);
    }
    if (g_app_ctx != NULL) {
        (*env)->DeleteGlobalRef(env, g_app_ctx);
    }
    g_app_ctx = (*env)->NewGlobalRef(env, ctx);

    jclass ctx_cls = (*env)->GetObjectClass(env, ctx);
    jmethodID get_cl = (*env)->GetMethodID(env, ctx_cls, "getClassLoader",
                                           "()Ljava/lang/ClassLoader;");
    jobject cl = (*env)->CallObjectMethod(env, ctx, get_cl);
    if (g_class_loader != NULL) {
        (*env)->DeleteGlobalRef(env, g_class_loader);
    }
    g_class_loader = (*env)->NewGlobalRef(env, cl);

    jclass cl_cls = (*env)->FindClass(env, "java/lang/ClassLoader");
    g_load_class = (*env)->GetMethodID(env, cl_cls, "loadClass",
                                       "(Ljava/lang/String;)Ljava/lang/Class;");

    (*env)->DeleteLocalRef(env, ctx_cls);
    (*env)->DeleteLocalRef(env, cl);
    (*env)->DeleteLocalRef(env, cl_cls);

    ASVC_LOGI("jni bridge initialized");
    androidsvcjni_on_init();
}

JNIEXPORT void JNICALL
Java_io_gosicp_androidsvc_GoForegroundService_nativeOnCreate(JNIEnv *env, jobject thiz) {
    jobject gref = (*env)->NewGlobalRef(env, thiz);
    androidsvcjni_on_create((jlong)(uintptr_t)gref);
}

JNIEXPORT jint JNICALL
Java_io_gosicp_androidsvc_GoForegroundService_nativeOnStartCommand(JNIEnv *env, jobject thiz, jstring action) {
    (void)thiz;
    char *cs;
    if (action != NULL) {
        const char *raw = (*env)->GetStringUTFChars(env, action, NULL);
        cs = strdup(raw);
        (*env)->ReleaseStringUTFChars(env, action, raw);
    } else {
        cs = strdup("");
    }
    int rc = androidsvcjni_on_start(cs);
    free(cs);
    return (jint)rc;
}

JNIEXPORT void JNICALL
Java_io_gosicp_androidsvc_GoForegroundService_nativeOnDestroy(JNIEnv *env, jobject thiz) {
    (void)env;
    (void)thiz;
    androidsvcjni_on_destroy();
}
