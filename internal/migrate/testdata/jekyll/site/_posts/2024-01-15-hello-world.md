---
title: Hello World
categories: [news]
tags: go jekyll
---
Intro paragraph for {{ page.title }} on {{ site.title }}.

<!--excerpt-->

* TOC
{:toc}

## Setup {#setup}

{% highlight ruby linenos %}
def hello
  puts "hi"
end
{% endhighlight %}

## Usage
{: #usage .big}

A notice.
{: .notice}

See [the docs](http://example.org){:target="_blank"} and [jump](#setup).

Read the [second post]({% post_url 2024-02-01-Second-Post %}) and the [about page]({% link about.md %}).
Also [contact](../contact.md) and the [recipe]({% link _recipes/pancakes.md %}).

![Logo]({{ site.baseurl }}/assets/img/logo.png)
![Logo again]({{ "/assets/img/logo.png" | relative_url }})

{% include note.html text="Remember this" %}

{% gist 123abc %}

{% raw %}
Literal {{ liquid }} stays.
{% endraw %}

{% comment %}
hidden
{% endcomment %}

{% if site.show_extra %}Extra text{% endif %}

```liquid
{{ site.title }}
```
