# Types with no-copy decoding

This sub-package contains a set of typed sclices which can be useful for encoding/decoding large numerical slices faster. This is relatively unsafe and non-portable as the encoding simply copies the memory of the slice, hence disregarding byte order of the encoder/decoders. However, this lets us to avoid allocating and copying memory when encoding/decoding, making this at least 10x faster than the safe implementation. 

# Warning

This implementation simply maps the byte slice provided in `Unmarshal` call to the Go structs which need to be decoded. This simply reuses the underlying byte array to store the data and *does not perform a memory copy*. This can be dangerous in many cases, `be careful how this is used`!

# Benchmark

Array of 10K elements:
```
BenchmarkUint64s_Safe/marshal-8         	   50000	     22914 ns/op	    2272 B/op	       5 allocs/op
BenchmarkUint64s_Safe/unmarshal-8       	   50000	     33724 ns/op	    4129 B/op	       2 allocs/op
BenchmarkUint64s_Unsafe/marshal-8       	 1000000	      2224 ns/op	    4977 B/op	       2 allocs/op
BenchmarkUint64s_Unsafe/unmarshal-8     	 5000000	       365 ns/op	      32 B/op	       1 allocs/op
```

# Usage
This is a drop-in type, so simply use one of the types available in the package (`Bools`, `Int32s`, `Uint64s` ...) and `Marshal` or `Unmarshal` using the binary package.
```
// Marshal some numbers
v := nocopy.Int32s{4, 5, 6, 1, 2, 3}
encoded, err := binary.Marshal(&v)

// Unmarshal the numbers
var o nocopy.Int32s
err = binary.Unmarshal(encoded, &o)
```
