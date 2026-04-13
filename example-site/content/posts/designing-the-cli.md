+++
title = "Designing a Two-Command CLI"
date = 2026-04-12T14:00:00Z
draft = false
tags = ["cli", "workflow"]
categories = ["product"]
description = "Why Nida keeps the public interface intentionally narrow."
slug = "designing-the-cli"
+++

# Designing a Two-Command CLI

Nida intentionally exposes only two user-facing commands:

- `nida build`
- `nida serve`

That constraint shapes the rest of the product. Instead of asking the user to assemble a workflow from many subcommands, the tool keeps the common loop direct: build the site, or work on it locally.

There is still room for flags where they help, but the core mental model remains tiny.
