package jni

// Value is a typed argument passed to NewObject / Call*. Build via the
// constructors (IntValue, ObjectValue, ...) — the zero value is invalid.
//
// The on-disk representation is platform-specific; the public surface is
// the constructors and their use as variadic arguments.
type Value struct {
	kind valueKind
	bits int64 // primitive bits (re-interpreted per kind)
	objR Object
}

type valueKind uint8

const (
	kindZero valueKind = iota
	kindInt
	kindLong
	kindBool
	kindFloat
	kindDouble
	kindObject
)

func IntValue(i int32) Value  { return Value{kind: kindInt, bits: int64(i)} }
func LongValue(i int64) Value { return Value{kind: kindLong, bits: i} }
func BoolValue(b bool) Value {
	v := int64(0)
	if b {
		v = 1
	}
	return Value{kind: kindBool, bits: v}
}
func FloatValue(f float32) Value  { return Value{kind: kindFloat, bits: int64(floatBits(f))} }
func DoubleValue(f float64) Value { return Value{kind: kindDouble, bits: doubleBits(f)} }
func ObjectValue(o Object) Value  { return Value{kind: kindObject, objR: o} }
