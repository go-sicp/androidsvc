package jni

import "math"

func floatBits(f float32) uint32 { return math.Float32bits(f) }
func doubleBits(f float64) int64 { return int64(math.Float64bits(f)) }
