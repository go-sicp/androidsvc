package jni

import "errors"

// ErrNotReady is returned by Do when the JNI bridge has not been initialized
// yet. The bridge is initialized once the Java side calls
// GoServiceBridge.init(Context), which triggers nativeInit on the cgo side.
var ErrNotReady = errors.New("androidsvc/jni: bridge not initialized (call GoServiceBridge.init from Java)")

// ErrJavaException wraps a Java exception that was thrown during a JNI call
// and cleared from the pending state by this bridge.
type ErrJavaException struct {
	Class   string
	Message string
}

func (e *ErrJavaException) Error() string {
	if e.Class == "" {
		return "java exception: " + e.Message
	}
	return e.Class + ": " + e.Message
}

// Class is an opaque handle to a java.lang.Class. It is only valid for the
// duration of the enclosing Do call unless converted to a GlobalRef.
type Class interface{ isClass() }

// Object is an opaque handle to a Java object reference (jobject).
type Object interface{ isObject() }

// MethodID is an opaque handle to a JNI method ID.
type MethodID interface{ isMethodID() }

// GlobalRef is a retained Java reference. Always call Release when the
// reference is no longer needed; otherwise the object is leaked.
type GlobalRef interface {
	Object
	Release()
}

// Env exposes the subset of JNIEnv operations needed by androidsvc. All
// returned handles are local references valid only for the duration of the
// enclosing Do call.
type Env interface {
	FindClass(name string) (Class, error)
	GetMethodID(c Class, name, sig string) (MethodID, error)
	GetStaticMethodID(c Class, name, sig string) (MethodID, error)

	NewObject(c Class, ctor MethodID, args ...Value) (Object, error)
	CallVoid(o Object, m MethodID, args ...Value) error
	CallObject(o Object, m MethodID, args ...Value) (Object, error)
	CallInt(o Object, m MethodID, args ...Value) (int32, error)
	CallStaticVoid(c Class, m MethodID, args ...Value) error
	CallStaticObject(c Class, m MethodID, args ...Value) (Object, error)

	NewString(s string) Object
	GetString(o Object) string

	NewGlobalRef(o Object) GlobalRef

	// AppContext returns the captured android.content.Context as a stable
	// global reference. The handle is valid across Do calls.
	AppContext() Object
}
