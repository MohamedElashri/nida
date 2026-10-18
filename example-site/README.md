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
* site templates in `templates/` and static assets in `static/`
* two demo themes that the site does not select: `themes/base/` and
  `themes/ink/`, which extends `base`. Add `theme = "ink"` to `config.toml`
  to try them; site templates still override the theme's.

Typical local usage:

```bash
go run ./cmd/nida build --site ./example-site
go run ./cmd/nida serve --site ./example-site
```
