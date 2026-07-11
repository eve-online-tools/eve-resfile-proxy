package transform

// WASM ABI:
//
// Guest exports:
//
//	transform(in_ptr i32, in_len i32, out_ptr i32, out_max i32) -> i32
//
// Returns the number of bytes written to out_ptr, or a negative error code.
//
// Optional host imports (module "env"):
//
//	get_res_path(buf_ptr i32, max_len i32) -> i32
//	get_cdn_path(buf_ptr i32, max_len i32) -> i32
//
// get_res_path and get_cdn_path write UTF-8 path bytes into guest memory when
// buf_ptr is non-zero and max_len is sufficient. They return the path length,
// or -1 when the buffer is too small.
