// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package nocopy

import (
	"unsafe"
)

func binaryToBools(b *[]byte) Bools {
	return *(*Bools)(unsafe.Pointer(b))
}

func boolsToBinary(v *Bools) []byte {
	return *(*[]byte)(unsafe.Pointer(v))
}
