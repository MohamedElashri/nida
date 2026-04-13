# Example Site

`example-site` is the bundled showcase website for `nida`.

It exists for two purposes:

* a realistic example project for new users
* a stable integration fixture for the test suite

The site includes:

* a complete `config.toml`
* multiple blog posts
* standalone pages
* tags and categories
* a lightweight built-in theme
* copied static assets

Typical local usage:

```bash
go run ./cmd/nida build --site ./example-site
go run ./cmd/nida serve --site ./example-site
```
