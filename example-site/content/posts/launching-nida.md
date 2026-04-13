+++
title = "Launching Nida"
date = 2026-04-13T09:30:00Z
draft = false
tags = ["architecture", "release"]
categories = ["engineering"]
description = "A first look at the publishing model Nida is built around."
slug = "launching-nida"
+++

# Launching Nida

Nida starts from a simple belief: many personal sites do not need a giant abstraction layer.

Instead of aiming for maximum flexibility, it tries to make a focused publishing workflow feel calm and obvious. The source files are plain. The outputs are deterministic. The command surface stays small enough to remember.

## What this first version includes

- Markdown content with TOML front matter
- syntax-highlighted fenced code blocks
- built-in tags and categories
- RSS and sitemap generation
- a local development server with rebuilds

```go
func main() {
    os.Exit(cli.Run(os.Args[1:]))
}
```

The interesting work happens behind that tiny entrypoint: config loading, content discovery, rendering, output writing, and local serving all move through the same internal pipeline.
