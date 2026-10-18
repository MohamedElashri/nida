---
Title: First Post
date: 2024-03-05T09:00:00+01:00
lastmod: 2024-04-01
tags: [go, hugo]
aliases: [old-first, /legacy/first/]
images: [cover.png]
summary: The very first post.
---
Intro with a [link to the second]({{< ref "second" >}}) and an [anchor]({{< relref "/posts/2024/nested.md#part" >}}).

{{< figure src="/images/pic.png" title="A picture" caption="Caption" target="_blank" >}}

{{< youtube id="abc123" autoplay="true" >}}

{{% notice warning %}}
Careful **now**.
{{% /notice %}}

{{< highlight go "linenos=table,hl_lines=2 3-4,linenostart=10,style=monokai" >}}
package main

func main() {}
{{< /highlight >}}

{{< details summary="More info" open=true >}}
Hidden text.
{{< /details >}}

Escaped: {{</* figure src="x" */>}}

Param: {{< param "author" >}} and {{< param "socialLinks.github" >}}.

```
{{</* youtube "in-code" */>}}
{{< youtube "ran-in-code" >}}
```

<span>raw html</span>

<!--more-->
Rest.
