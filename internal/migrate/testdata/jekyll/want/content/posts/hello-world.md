---
title: Hello World
categories:
  - news
tags:
  - go
  - jekyll
comments: true
date: "2024-01-15"
template: post
path: /news/2024/01/15/hello-world.html
---
Intro paragraph for Hello World on Jekyll Fixture.

<!-- more -->

## Setup {#setup}

```ruby {linenos=true}
def hello
  puts "hi"
end
```

## Usage {#usage .big}

A notice.

See [the docs](http://example.org) and [jump](#setup).

Read the [second post](@/posts/Second-Post.md) and the [about page](@/about.md).
Also [contact](@/contact.md) and the [recipe](@/recipes/pancakes.md).

![Logo](/assets/img/logo.png)
![Logo again](/assets/img/logo.png)

{{< note text="Remember this" >}}

{{< gist "123abc" >}}

Literal {{ liquid }} stays.

{% if site.show_extra %}Extra text{% endif %}

```liquid
{{ site.title }}
```
