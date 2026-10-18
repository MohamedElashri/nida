+++
title = "Shortcodes"
date = 2024-01-02
# a comment that must survive
slug = "short-codes"
aliases = ["/old/shortcodes/"]
+++
Inline: {{ youtube(id="dQw4w9WgXcQ", autoplay=true) }}

{{ gallery(images=["a.png", "b.png"], width=300) }}

{% note(kind="warning") %}
Inside a **block** shortcode.
{% end %}

{% details(summary="More") %}
Native details block.
{% end %}

Escaped: {{/* youtube(id="x") */}}

Template leftover: {{ page.title }}

```
{{ youtube(id="in-code") }}
```

And `{{ youtube(id="inline-code") }}` too.

<div class="raw">raw html</div>
