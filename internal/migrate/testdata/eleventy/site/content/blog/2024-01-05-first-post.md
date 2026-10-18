---
title: This is my first post.
description: This is a post on My Blog about agile frameworks.
tags:
  - another tag
  - posts
---
Leverage agile frameworks to provide a robust synopsis for high level overviews.

<!-- excerpt -->

{% image "./possum.png", "A possum parent and two possum kids hanging from the iconic red balloon" %}

{% aside "note" %}
Bring to the table **win-win** survival strategies.
{% endaside %}

The site title is {{ metadata.title }} and this post is {{ title }}.

{% highlight "js 2" %}
function myFunction() {
  return true;
}
{% endhighlight %}

{% raw %}
```liquid
{{ this stays literal }}
```
{% endraw %}

```js
// Eleventy ran this as a template
let x = "{{ page.url }}";
```

Built on {% currentBuildDate %}.

{% for item in collections.posts %}- {{ item.data.title }}{% endfor %}
