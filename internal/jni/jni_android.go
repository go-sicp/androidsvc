//go:build android

package jni

/*
#cgo LDFLAGS: -llog

#include <jni.h>
#include <stdlib.h>
#include <string.h>
#include <android/log.h>

// Globals are defined exactly once in jni_native_android.c. The cgo
// preamble re-emits its #include directives in every compilation unit it
// appears in; defining variables here would produce duplicate symbols.
extern JavaVM   *g_vm;
extern jobject   g_app_ctx;
extern jobject   g_class_loader;
extern jmethodID g_load_class;

// ----- helpers callable from Go cgo -----

static int asvc_attach(JNIEnv **env, int *out_attached) {
    *out_attached = 0;
    if (g_vm == NULL) return -1;
    int rc = (*g_vm)->GetEnv(g_vm, (void**)env, JNI_VERSION_1_6);
    if (rc == JNI_OK) return 0;
    if (rc == JNI_EDETACHED) {
        rc = (*g_vm)->AttachCurrentThread(g_vm, env, NULL);
        if (rc != 0) return rc;
        *out_attached = 1;
        return 0;
    }
    return rc;
}

static void asvc_detach(void) {
    if (g_vm != NULL) (*g_vm)->DetachCurrentThread(g_vm);
}

static int asvc_push_frame(JNIEnv *env, int cap) {
    return (*env)->PushLocalFrame(env, cap);
}

static void asvc_pop_frame(JNIEnv *env) {
    (*env)->PopLocalFrame(env, NULL);
}

static jobject asvc_app_ctx(void) { return g_app_ctx; }

static int asvc_ready(void) {
    return g_vm != NULL && g_app_ctx != NULL;
}

// FindClass via ClassLoader.loadClass to be safe off the main thread.
static jclass asvc_find_class(JNIEnv *env, const char *slash_name) {
    if (g_class_loader == NULL || g_load_class == NULL) {
        return (*env)->FindClass(env, slash_name);
    }
    char *dotted = strdup(slash_name);
    if (dotted == NULL) return NULL;
    for (char *p = dotted; *p; p++) {
        if (*p == '/') *p = '.';
    }
    jstring s = (*env)->NewStringUTF(env, dotted);
    free(dotted);
    jobject cls = (*env)->CallObjectMethod(env, g_class_loader, g_load_class, s);
    (*env)->DeleteLocalRef(env, s);
    return (jclass)cls;
}

static jmethodID    asvc_get_method(JNIEnv *e, jclass c, const char *n, const char *s) { return (*e)->GetMethodID(e, c, n, s); }
static jmethodID    asvc_get_smethod(JNIEnv *e, jclass c, const char *n, const char *s){ return (*e)->GetStaticMethodID(e, c, n, s); }
static jobject      asvc_new_obj(JNIEnv *e, jclass c, jmethodID m, jvalue *a)          { return (*e)->NewObjectA(e, c, m, a); }
static void         asvc_call_v(JNIEnv *e, jobject o, jmethodID m, jvalue *a)          { (*e)->CallVoidMethodA(e, o, m, a); }
static jobject      asvc_call_o(JNIEnv *e, jobject o, jmethodID m, jvalue *a)          { return (*e)->CallObjectMethodA(e, o, m, a); }
static jint         asvc_call_i(JNIEnv *e, jobject o, jmethodID m, jvalue *a)          { return (*e)->CallIntMethodA(e, o, m, a); }
static void         asvc_scall_v(JNIEnv *e, jclass c, jmethodID m, jvalue *a)          { (*e)->CallStaticVoidMethodA(e, c, m, a); }
static jobject      asvc_scall_o(JNIEnv *e, jclass c, jmethodID m, jvalue *a)          { return (*e)->CallStaticObjectMethodA(e, c, m, a); }
static jstring      asvc_new_str(JNIEnv *e, const char *s)                             { return (*e)->NewStringUTF(e, s); }
static const char  *asvc_str_get(JNIEnv *e, jstring s)                                 { return (*e)->GetStringUTFChars(e, s, NULL); }
static void         asvc_str_rel(JNIEnv *e, jstring s, const char *cs)                 { (*e)->ReleaseStringUTFChars(e, s, cs); }
static jobject      asvc_global(JNIEnv *e, jobject o)                                  { return (*e)->NewGlobalRef(e, o); }
static void         asvc_dglobal(JNIEnv *e, jobject o)                                 { (*e)->DeleteGlobalRef(e, o); }
static void         asvc_dlocal(JNIEnv *e, jobject o)                                  { (*e)->DeleteLocalRef(e, o); }

static jthrowable asvc_exc_get(JNIEnv *e) {
    jthrowable t = (*e)->ExceptionOccurred(e);
    if (t != NULL) (*e)->ExceptionClear(e);
    return t;
}
*/
import "C"

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// --- ready / init plumbing ---------------------------------------------------

var (
	ready atomic.Bool

	initMu     sync.Mutex
	initDone   chan struct{} = make(chan struct{})
	initClosed bool

	svcMu      sync.RWMutex
	svcCreate  func(int64)
	svcStart   func(string) int32
	svcDestroy func()
)

//export androidsvcjni_on_init
func androidsvcjni_on_init() {
	ready.Store(true)
	initMu.Lock()
	if !initClosed {
		close(initDone)
		initClosed = true
	}
	initMu.Unlock()
}

//export androidsvcjni_on_create
func androidsvcjni_on_create(svcRef C.jlong) {
	svcMu.RLock()
	fn := svcCreate
	svcMu.RUnlock()
	if fn != nil {
		fn(int64(svcRef))
	}
}

//export androidsvcjni_on_start
func androidsvcjni_on_start(action *C.char) C.int {
	a := C.GoString(action)
	svcMu.RLock()
	fn := svcStart
	svcMu.RUnlock()
	if fn == nil {
		return C.int(2) // START_NOT_STICKY
	}
	return C.int(fn(a))
}

//export androidsvcjni_on_destroy
func androidsvcjni_on_destroy() {
	svcMu.RLock()
	fn := svcDestroy
	svcMu.RUnlock()
	if fn != nil {
		fn()
	}
}

// IsReady reports whether the bridge has captured the JavaVM and Context.
func IsReady() bool { return ready.Load() }

// AdoptGlobal wraps an existing JNI global ref (jobject cast to int64 via
// uintptr) into a GlobalRef without creating a new global ref. Used to
// hand off refs created by the cgo trampolines in this package.
func AdoptGlobal(handle int64) GlobalRef {
	if handle == 0 {
		return &globalRef{}
	}
	return &globalRef{raw: *(*C.jobject)(unsafe.Pointer(&handle))}
}

// SetServiceCallbacks registers the lifecycle callbacks the cgo trampolines
// will invoke. Pass nil to clear.
func SetServiceCallbacks(create func(int64), start func(string) int32, destroy func()) {
	svcMu.Lock()
	svcCreate, svcStart, svcDestroy = create, start, destroy
	svcMu.Unlock()
}

// --- Do ----------------------------------------------------------------------

// Do attaches the current goroutine's OS thread to the JVM, pushes a fresh
// JNI local-ref frame, runs fn, then pops the frame and (if it attached the
// thread) detaches.
func Do(fn func(Env) error) error {
	if !ready.Load() {
		return ErrNotReady
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var env *C.JNIEnv
	var attached C.int
	if rc := C.asvc_attach(&env, &attached); rc != 0 {
		return ErrNotReady
	}
	if attached != 0 {
		defer C.asvc_detach()
	}
	if rc := C.asvc_push_frame(env, 64); rc != 0 {
		return &ErrJavaException{Message: "PushLocalFrame failed"}
	}
	defer C.asvc_pop_frame(env)

	e := &androidEnv{raw: env}
	return fn(e)
}

// Typed zero-value sentinels. NDK r27's `typedef void* jobject` becomes a
// non-pointer Go cgo type, so comparing C handles to untyped `nil` fails to
// compile. We compare against these instead.
var (
	asvcNilJObj C.jobject
	asvcNilJCls C.jclass
	asvcNilJStr C.jstring
	asvcNilJThr C.jthrowable
	asvcNilJMid C.jmethodID
)

// --- handle types ------------------------------------------------------------

type androidEnv struct {
	raw *C.JNIEnv
}

type class struct{ raw C.jclass }

func (*class) isClass() {}

type object struct{ raw C.jobject }

func (*object) isObject() {}

type methodID struct{ raw C.jmethodID }

func (*methodID) isMethodID() {}

type globalRef struct {
	raw C.jobject
	rel sync.Once
}

func (*globalRef) isObject() {}
func (g *globalRef) Release() {
	g.rel.Do(func() {
		if g.raw == asvcNilJObj {
			return
		}
		_ = Do(func(e Env) error {
			ae := e.(*androidEnv)
			C.asvc_dglobal(ae.raw, g.raw)
			g.raw = asvcNilJObj
			return nil
		})
	})
}

// --- Env methods -------------------------------------------------------------

func (e *androidEnv) check() error {
	t := C.asvc_exc_get(e.raw)
	if t == asvcNilJThr {
		return nil
	}
	// Recover Throwable.toString(); failure here just yields a generic msg.
	tCls := C.asvc_find_class(e.raw, C.CString("java/lang/Throwable"))
	if tCls == asvcNilJCls {
		return &ErrJavaException{Message: "java exception (class lookup failed)"}
	}
	defer C.asvc_dlocal(e.raw, C.jobject(tCls))

	mid := C.asvc_get_method(e.raw, tCls,
		C.CString("toString"), C.CString("()Ljava/lang/String;"))
	if mid == asvcNilJMid {
		return &ErrJavaException{Message: "java exception"}
	}
	js := C.asvc_call_o(e.raw, C.jobject(t), mid, nil)
	if js == asvcNilJObj {
		return &ErrJavaException{Message: "java exception (toString failed)"}
	}
	defer C.asvc_dlocal(e.raw, js)
	cs := C.asvc_str_get(e.raw, C.jstring(js))
	defer C.asvc_str_rel(e.raw, C.jstring(js), cs)
	return &ErrJavaException{Message: C.GoString(cs)}
}

func (e *androidEnv) FindClass(name string) (Class, error) {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	c := C.asvc_find_class(e.raw, cs)
	if err := e.check(); err != nil {
		return nil, err
	}
	if c == asvcNilJCls {
		return nil, &ErrJavaException{Message: "class not found: " + name}
	}
	return &class{raw: c}, nil
}

func (e *androidEnv) GetMethodID(c Class, name, sig string) (MethodID, error) {
	cn := C.CString(name)
	cs := C.CString(sig)
	defer C.free(unsafe.Pointer(cn))
	defer C.free(unsafe.Pointer(cs))
	m := C.asvc_get_method(e.raw, c.(*class).raw, cn, cs)
	if err := e.check(); err != nil {
		return nil, err
	}
	if m == asvcNilJMid {
		return nil, &ErrJavaException{Message: "method not found: " + name + sig}
	}
	return &methodID{raw: m}, nil
}

func (e *androidEnv) GetStaticMethodID(c Class, name, sig string) (MethodID, error) {
	cn := C.CString(name)
	cs := C.CString(sig)
	defer C.free(unsafe.Pointer(cn))
	defer C.free(unsafe.Pointer(cs))
	m := C.asvc_get_smethod(e.raw, c.(*class).raw, cn, cs)
	if err := e.check(); err != nil {
		return nil, err
	}
	if m == asvcNilJMid {
		return nil, &ErrJavaException{Message: "static method not found: " + name + sig}
	}
	return &methodID{raw: m}, nil
}

func (e *androidEnv) packArgs(args []Value) (*C.jvalue, func()) {
	if len(args) == 0 {
		return nil, func() {}
	}
	arr := C.malloc(C.size_t(C.sizeof_jvalue * len(args)))
	jv := (*C.jvalue)(arr)
	for i, v := range args {
		v.set(jv, C.int(i))
	}
	return jv, func() { C.free(arr) }
}

func (e *androidEnv) NewObject(c Class, ctor MethodID, args ...Value) (Object, error) {
	jv, free := e.packArgs(args)
	defer free()
	o := C.asvc_new_obj(e.raw, c.(*class).raw, ctor.(*methodID).raw, jv)
	if err := e.check(); err != nil {
		return nil, err
	}
	if o == asvcNilJObj {
		return nil, &ErrJavaException{Message: "NewObject returned null"}
	}
	return &object{raw: o}, nil
}

func (e *androidEnv) CallVoid(o Object, m MethodID, args ...Value) error {
	jv, free := e.packArgs(args)
	defer free()
	C.asvc_call_v(e.raw, objRaw(o), m.(*methodID).raw, jv)
	return e.check()
}

func (e *androidEnv) CallObject(o Object, m MethodID, args ...Value) (Object, error) {
	jv, free := e.packArgs(args)
	defer free()
	r := C.asvc_call_o(e.raw, objRaw(o), m.(*methodID).raw, jv)
	if err := e.check(); err != nil {
		return nil, err
	}
	if r == asvcNilJObj {
		return nil, nil
	}
	return &object{raw: r}, nil
}

func (e *androidEnv) CallInt(o Object, m MethodID, args ...Value) (int32, error) {
	jv, free := e.packArgs(args)
	defer free()
	r := C.asvc_call_i(e.raw, objRaw(o), m.(*methodID).raw, jv)
	if err := e.check(); err != nil {
		return 0, err
	}
	return int32(r), nil
}

func (e *androidEnv) CallStaticVoid(c Class, m MethodID, args ...Value) error {
	jv, free := e.packArgs(args)
	defer free()
	C.asvc_scall_v(e.raw, c.(*class).raw, m.(*methodID).raw, jv)
	return e.check()
}

func (e *androidEnv) CallStaticObject(c Class, m MethodID, args ...Value) (Object, error) {
	jv, free := e.packArgs(args)
	defer free()
	r := C.asvc_scall_o(e.raw, c.(*class).raw, m.(*methodID).raw, jv)
	if err := e.check(); err != nil {
		return nil, err
	}
	if r == asvcNilJObj {
		return nil, nil
	}
	return &object{raw: r}, nil
}

func (e *androidEnv) NewString(s string) Object {
	cs := C.CString(s)
	defer C.free(unsafe.Pointer(cs))
	js := C.asvc_new_str(e.raw, cs)
	if js == asvcNilJStr {
		return nil
	}
	return &object{raw: C.jobject(js)}
}

func (e *androidEnv) GetString(o Object) string {
	if o == nil {
		return ""
	}
	js := C.jstring(objRaw(o))
	cs := C.asvc_str_get(e.raw, js)
	defer C.asvc_str_rel(e.raw, js, cs)
	return C.GoString(cs)
}

func (e *androidEnv) NewGlobalRef(o Object) GlobalRef {
	if o == nil {
		return &globalRef{}
	}
	g := C.asvc_global(e.raw, objRaw(o))
	return &globalRef{raw: g}
}

func (e *androidEnv) AppContext() Object {
	return &object{raw: C.asvc_app_ctx()}
}

// --- helpers -----------------------------------------------------------------

// objRaw extracts the underlying jobject for object or globalRef wrappers.
func objRaw(o Object) C.jobject {
	switch v := o.(type) {
	case *object:
		return v.raw
	case *globalRef:
		return v.raw
	}
	return asvcNilJObj
}
