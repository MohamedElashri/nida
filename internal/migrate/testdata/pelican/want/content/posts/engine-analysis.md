---
title: Analysis of the Engine
date: "2023-12-10T18:30:00+01:00"
description: The engine, in reStructuredText.
tags:
  - math
  - rst
categories:
  - Analysis
---
## Introduction

This is *emphasis*, **strong**, and `inline literal` text, with
a [link to Python](https://www.python.org/) and a reference to [Docutils](https://docutils.sourceforge.io/).
Inline math: $a^2 + b^2$. A footnote [^1] and &lt;b>angle brackets&lt;/b>.

- First item
- Second item with *markup*

  - Nested item

1. Numbered one
2. Numbered two

### Details

Here is a literal block:

```
def hello():
    return "world"
```

```python {linenos}
print("highlighted")
```

![The logo](/images/logo.png)

{{< figure src="/images/logo.png" caption="The engine logo." width="200" >}}

> [!NOTE]
> Notes become callouts.

> [!WARNING]
> A one-line warning.

| Input | Output |
| --- | --- |
| 1 | 2 |
| 3 | 4 |

```rst
.. graphviz::

   digraph { a -> b }
```

**Term**\
Its definition.

---

See also [the engine post](@/posts/the-difference-engine-revisited.md).

[^1]: The footnote text.
