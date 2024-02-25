//go:build !js
// +build !js

// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package binary

import (
	"reflect"
	"unsafe"
)

// ToString converts byte slice to a string without allocating.
func ToString(b *[]byte) string {
	return *(*string)(unsafe.Pointer(b))
}

// ToBytes converts a string to a byte slice without allocating.
func ToBytes(v string) []byte {
	strHeader := (*reflect.StringHeader)(unsafe.Pointer(&v))
	bytesData := unsafe.Slice((*byte)(unsafe.Pointer(strHeader.Data)), len(v))

	return bytesData
}

func binaryToBools(b *[]byte) []bool {
	return *(*[]bool)(unsafe.Pointer(b))
}

func boolsToBinary(v *[]bool) []byte {
	return *(*[]byte)(unsafe.Pointer(v))
}
