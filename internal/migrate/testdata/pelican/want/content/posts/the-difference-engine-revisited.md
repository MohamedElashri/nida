---
title: The Difference Engine, Revisited!
date: "2024-03-05T10:20:00+01:00"
updated: "2024-03-06T08:00:00Z"
description: A short look at the engine that was never finished.
tags:
  - babbage
  - engines
  - history
categories:
  - Machines
authors:
  - Ada Lovelace
  - Charles Babbage
series: engines
---
## Background {#background}

The engine computed polynomial tables[^1]. See [the analysis](@/posts/engine-analysis.md)
and [about me](@/pages/about.md#contact).

![Logo](/images/logo.png)
The [plan](/2024/03/diagrams/plan.svg) is attached.

```python
def difference(table):
    return [b - a for a, b in zip(table, table[1:])]
```

Inline `{filename}not-a-link.md` stays as it is.

[^1]: Tables of logarithms, mostly.
