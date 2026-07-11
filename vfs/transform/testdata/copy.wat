;; Identity transform for tests: copies input bytes to the output buffer.
;; ABI: transform(in_ptr, in_len, out_ptr, out_max) -> i32

(module
  (memory (export "memory") 1)

  (func (export "transform")
    (param $in_ptr i32) (param $in_len i32) (param $out_ptr i32) (param $out_max i32)
    (result i32)
    (local $i i32)

    (if (i32.ge_u (local.get $in_len) (local.get $out_max))
      (then (return (i32.const -1))))

    (local.set $i (i32.const 0))
    (block $done
      (loop $copy
        (if (i32.ge_u (local.get $i) (local.get $in_len))
          (then (br $done)))

        (i32.store8
          (i32.add (local.get $out_ptr) (local.get $i))
          (i32.load8_u (i32.add (local.get $in_ptr) (local.get $i))))

        (local.set $i (i32.add (local.get $i) (i32.const 1)))
        (br $copy)
      )
    )
    (local.get $in_len)
  )
)
