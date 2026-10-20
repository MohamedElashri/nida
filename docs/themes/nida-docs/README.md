# nida-docs

The documentation theme used by the Nida docs site. It shows how a docs theme
can be built from Nida's features: a sidebar from `.Index.Nav`, a pager from
`.Index.Neighbors`, breadcrumbs, an on-page outline from `.Page.TOC`, search,
dark mode, translations, and data files.

## Settings

| Setting | Where | Purpose |
| --- | --- | --- |
| `docs_root` | `[extra]` | Section whose tree fills the sidebar and pager. Defaults to the whole site. |
| `edit_url` | site config | Shows an "Edit this page" link. |
| `updated_from_git` | site config | Shows a "Last updated" date. |
| `[search] enabled` | site config | Enables the search dialog (Ctrl K or `/`). |
| `[markdown] math` | site config | Renders TeX to MathML at build time. With math on, every page gets Nida's bundled math stylesheet. Mermaid diagrams are rendered to SVG at build time and follow the light and dark color schemes. |

## Data files

The theme reads these from the site's `data/` directory. Every key is optional.

- `project.toml`: `name`, `version`, `license`, `repo`, `install` (header badge, footer, home install command).
- `nav.toml`: `[[top]]` header links and `[[footer]]` columns, each with `name` and `url`.
- `home.toml`: hero `eyebrow`, `title` (Markdown), `tagline`, `[[actions]]`, a code `sample` with `sample_title`, `[[features]]` cards with `icon`, `title`, `text`, and `items` (shown as tags), and an `about` section with `about_title`.

Feature card icons: `pen`, `layers`, `rss`, `bolt`, `check`, `globe`, and the
other names in `templates/partials/icons.html`.

## Shortcodes

| Shortcode | Use |
| --- | --- |
| `{{< tabs >}}{{< tab "Label" >}}...{{< /tab >}}{{< /tabs >}}` | Tabbed panels; the chosen label is remembered across pages. |
| `{{< cards >}}{{< card title="..." href="@/page.md" >}}Text{{< /card >}}{{< /cards >}}` | A grid of link cards. |
| `{{< path title="..." icon="pen" >}}1. [Step](@/page.md){{< /path >}}` | A card with a short reading list, used inside `cards`. |
| `{{< steps >}}...{{< /steps >}}` | Numbers each `###` heading inside as a step. |
| `{{< since "0.7.0" >}}` | A "New in" version badge, for headings or text. |
| `{{< kbd "Ctrl+K" >}}` | Keyboard keys. |

## Translations

UI strings live in `i18n/en.toml`. A site can translate them by adding
`i18n/<language>.toml` with the same keys.
