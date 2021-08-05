## Benchmarks

These are the outputs from the chunkenc appender and iterator benchmarks.

Types of benchmarks
* Same: Is always appending and iterating over the same value `100`. 
  * Run-Length-Encoding (RLE) is supposed to be the best.
* Increasing: Is always appending and iterating over the value increasing it by `100` each time.
  * Double-Delta-Encoding (Delta) is supposed to be the best.
* Random: Is always appending and iterating random values between `0` and `1_000_000`.
  * XOR is supposed to be the best.
  