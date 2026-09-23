//go:build android

package jni

/*
#include <jni.h>

// asvc_set_* pack a Go-side Value into a jvalue cell. They live in this file
// (rather than in jni_android.go) because cgo `static` C helpers are
// file-local to the .o and therefore not visible across cgo files within
// the same Go package.
static void asvc_set_int   (jvalue *arr, int i, jint v)    { arr[i].i = v; }
static void asvc_set_long  (jvalue *arr, int i, jlong v)   { arr[i].j = v; }
static void asvc_set_bool  (jvalue *arr, int i, jboolean v){ arr[i].z = v; }
static void asvc_set_float (jvalue *arr, int i, jfloat v)  { arr[i].f = v; }
static void asvc_set_double(jvalue *arr, int i, jdouble v) { arr[i].d = v; }
static void asvc_set_obj   (jvalue *arr, int i, jobject v) { arr[i].l = v; }
*/
import "C"

import "math"

// set writes v into arr[i] using the JNI jvalue layout. Called by Env when
// packing a []Value into a contiguous jvalue array for the Call*MethodA family.
func (v Value) set(arr *C.jvalue, i C.int) {
	switch v.kind {
	case kindInt:
		C.asvc_set_int(arr, i, C.jint(int32(v.bits)))
	case kindLong:
		C.asvc_set_long(arr, i, C.jlong(v.bits))
	case kindBool:
		var b C.jboolean
		if v.bits != 0 {
			b = 1
		}
		C.asvc_set_bool(arr, i, b)
	case kindFloat:
		C.asvc_set_float(arr, i, C.jfloat(math.Float32frombits(uint32(v.bits))))
	case kindDouble:
		C.asvc_set_double(arr, i, C.jdouble(math.Float64frombits(uint64(v.bits))))
	case kindObject:
		C.asvc_set_obj(arr, i, objRaw(v.objR))
	default:
		// kindZero: leave the slot as written by malloc (caller used
		// C.malloc, contents are uninitialized — but JNI never reads
		// unused slots, so this is fine).
	}
}
