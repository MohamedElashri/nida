//go:build amd64 || arm64 || mips64 || mips64le || ppc64 || ppc64le || riscv64 || s390x || wasm
// +build amd64 arm64 mips64 mips64le ppc64 ppc64le riscv64 s390x wasm

/*
Package fast contains code ported from V8 (https://github.com/v8/v8/blob/master/src/numbers/fast-dtoa.cc)

See LICENSE_V8 for the original copyright message and disclaimer.
*/
package fast

import "errors"

var (
	dcheckFailure = errors.New("DCHECK assertion failed")
)

func _DCHECK(f bool) {
	if !f {
		panic(dcheckFailure)
	}
}
