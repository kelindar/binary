# Unsafe Binary Slices

This sub-package contains a set of typed sclices which can be useful for encoding/decoding large numerical slices faster. This is relatively unsafe and non-portable as the encoding simply copies the memory of the slice, hence disregarding byte order of the encoder/decoders. However, this lets us to avoid allocating and copying memory when encoding/decoding, making this at least 10x faster than the safe implementation. 


# Benchmark

```
BenchmarkUint64s_Safe/marshal-8                   200000              9958 ns/op            1120 B/op          4 allocs/op
BenchmarkUint64s_Safe/unmarshal-8                 100000             16036 ns/op            4224 B/op          4 allocs/op
BenchmarkUint64s_Unsafe/marshal-8                1000000              1261 ns/op            4209 B/op          2 allocs/op
BenchmarkUint64s_Unsafe/unmarshal-8              1000000              1559 ns/op            4224 B/op          4 allocs/op
```

# Usage
This is a drop-in type, so simply use one of the types available in the package (`Bools`, `Int32s`, `Uint64s` ...) and `Marshal` or `Unmarshal` using the binary package.
```
// Marshal some numbers
v := Int32s{4, 5, 6, 1, 2, 3}
encoded, err := binary.Marshal(&v)

// Unmarshal the numbers
var o Int32s
err = binary.Unmarshal(b, &o)
```
