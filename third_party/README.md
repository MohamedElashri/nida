# Third-party code

Nida runs two JavaScript libraries at build time, Pele (Mermaid diagrams to
SVG) and Temml (TeX to MathML), in goja, a JavaScript engine written in Go.
This directory pins all three. Edit the pins here, never the generated files.

## Pele and Temml: `js/`

`js/package.json` and `js/package-lock.json` pin the npm packages. They are
not installed: nothing here runs on Node. They exist so Dependabot can watch
the versions.

`make vendor-js` (`go run ./scripts/vendorjs`) downloads each tarball named
in the lockfile, checks it against the lockfile's integrity hash, and writes:

| File | From |
| --- | --- |
| `internal/diagram/files/pele.global.js.gz` | `pele/dist/pele.global.js`, gzipped |
| `internal/mathml/files/temml.min.js.gz` | `temml/dist/temml.min.js`, gzipped |
| `internal/bundled/files/temml/` | Temml's stylesheets, `Temml.woff2`, and license |
| `internal/bundled/version.go` | `TemmlVersion` |

Latin Modern Math and its GUST Font License come from no npm package, so
`scripts/vendorjs` pins them by URL and SHA-256 and downloads them only when
the files on disk do not match.

To update by hand:

```sh
cd third_party/js && npm install --package-lock-only --save-exact temml@0.15.0 && cd ../..
make vendor-js
go test ./...
```

For a Dependabot pull request, check out its branch, run `make vendor-js`,
and push. CI runs `go run ./scripts/vendorjs -check` and fails until then.
Read the library's changelog: a new Pele release can change diagram layout,
and a new Temml release can change MathML output or error messages that
tests expect.

## goja: `goja/` and `goja.patch`

`goja/` is upstream goja at the version `go.mod` requires, without tests,
with `goja.patch` applied. `go.mod` points goja here with a `replace`.

The patch, described at its top:

- makes `String.prototype.localeCompare` compare code points instead of
  using `golang.org/x/text/collate`. Neither library calls `localeCompare`,
  and the collation tables would add 1.3 MB to the binary.
- fixes integer conversions that CodeQL flags and that go wrong where `int`
  is 32 bits, such as a `toString` radix that truncates before its range
  check.

`scripts/vendorgoja` then adds a build constraint to every file so goja
builds only on 64-bit platforms (amd64, arm64, mips64, ppc64, riscv64,
s390x, and wasm), and writes `nida_requires_64bit.go`, which makes a build
anywhere else fail with `undefined: nida_requires_a_64_bit_platform`. goja
converts `int64` to `int` throughout, which only loses data where
`int` is 32 bits. Each file carries both `//go:build` and `// +build`
lines, because CodeQL reads only the second. `loong64` is left out because
CodeQL does not know it, and would then ignore the whole constraint.

CodeQL still reports goja's ECMAScript `ToInt8`, `ToUint8`, `ToInt16`,
`ToUint16`, and `ToInt32` functions and its typed-array stores, which
convert to smaller integer types on purpose: the specification requires
the value to wrap around. Dismiss those alerts as false positives rather
than patching them.

Dependabot cannot update a replaced module, so it ignores goja. Instead
`.github/workflows/goja.yml` checks upstream every Monday, tries the patch,
and opens or updates an "Update vendored goja" issue. To update:

```sh
make goja-update               # or GOJA=<commit or pseudo-version>
go test ./...
```

If the patch no longer applies, edit `goja.patch` against the new upstream
files (it is a plain unified diff with `a/` and `b/` paths, made without the
generated build constraints) and run the command again. Before changing the
patch, run goja's own tests and the TC39 test262 suite against it: copy the
upstream module with its tests, `git apply` the patch, run
`.tc39_test262_checkout.sh`, then `go test ./...`. CI runs `go run ./scripts/vendorgoja -check`, which fails if
`goja/` is edited by hand.
