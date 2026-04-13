# Example Arabic Site

`example-site-ar` is a bundled Arabic showcase for `nida`.

It exists to show how a single-language Arabic blog can use the same simple workflow:

* `language = "ar"` for RTL output
* Markdown content in Arabic
* theme templates with Arabic labels
* the same `build` and `serve` commands as any other Nida site

Typical local usage:

```bash
go run ./cmd/nida build --site ./example-site-ar
go run ./cmd/nida serve --site ./example-site-ar
```
