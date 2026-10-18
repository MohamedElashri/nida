---
title: First Post
date: 2024-03-05T09:00:00+01:00
updated: 2024-04-01
tags: [go, hugo]
aliases:
  - /posts/old-first
  - /legacy/first/
images: [cover.png]
description: The very first post.
image: cover.png
---
Intro with a [link to the second](@/posts/second.md) and an [anchor](@/posts/2024/nested.md#part).

{{< figure src="/images/pic.png" caption="A picture: Caption" >}}

{{< youtube id="abc123" >}}

{{< notice "warning" >}}
Careful **now**.
{{< /notice >}}

```go {linenos=true, hl_lines="2 3-4", linenostart=10}
package main

func main() {}
```

{% details(summary="More info") %}
Hidden text.
{% end %}

Escaped: &#123;{< figure src="x" >}}

Param: Ada Lovelace and https://github.com/ada.

```
{{< youtube "in-code" >}}
{{< youtube "ran-in-code" >}}
```

<span>raw html</span>

<!--more-->
Rest.
