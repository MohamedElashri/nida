+++
title = "Nida Docs"
description = "A small, fast Go static site generator for blogs, personal sites, and documentation."
sort_by = "weight"
+++

## Choose your path

{{< cards >}}
{{< path title="Start a blog" icon="pen" >}}
1. [Install and create a site](@/guide/getting-started.md)
2. [Write posts](@/guide/writing.md) with summaries and tags
3. [Publish it](@/guide/deployment.md)

`nida init ./blog`
{{< /path >}}
{{< path title="Build a docs site" icon="layers" >}}
1. [Organize sections](@/reference/front-matter.md#section-fields) with weights
2. [Generate navigation](@/guide/templates.md#navigation-for-docs-sites)
3. [Add search](@/reference/search.md)

`nida check --external`
{{< /path >}}
{{< path title="Move an existing site" icon="arrow-right" >}}
1. [Map your old generator](@/guide/migrating.md)
2. [Keep your URLs](@/reference/config.md#permalinks)
3. [Fix what check reports](@/reference/cli.md#check)

`nida check --site .`
{{< /path >}}
{{< path title="Make a theme" icon="edit" >}}
1. [Learn the override order](@/guide/themes.md#override-order)
2. [Start from a minimal theme](@/guide/themes.md#a-minimal-theme)
3. [Use the template functions](@/guide/templates.md#template-functions)

`theme = "my-theme"`
{{< /path >}}
{{< /cards >}}
