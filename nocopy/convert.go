// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package nocopy

import (
	"reflect"
	"unsafe"
)

func binaryToString(b *[]byte) string {
	return *(*string)(unsafe.Pointer(b))
}

func stringToBinary(v string) (b []byte) {
	strHeader := (*reflect.StringHeader)(unsafe.Pointer(&v))
	byteHeader := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	byteHeader.Data = strHeader.Data

	l := len(v)
	byteHeader.Len = l
	byteHeader.Cap = l
	return
}

func binaryToBools(b *[]byte) Bools {
	return *(*Bools)(unsafe.Pointer(b))
}

func boolsToBinary(v *Bools) []byte {
	return *(*[]byte)(unsafe.Pointer(v))
}
