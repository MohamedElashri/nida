Title: The Difference Engine, Revisited!
Date: 2024-03-05 10:20
Modified: 2024-03-06T08:00:00+00:00
Category: Machines
Tags: babbage, engines, history
Authors: Ada Lovelace; Charles Babbage
Summary: A short look at the engine
    that was never finished.
Series: engines

[TOC]

## Background {: #background }

The engine computed polynomial tables[^1]. See [the analysis]({filename}../notes/math.rst)
and [about me]({filename}/pages/about.md#contact).

![Logo]({static}/images/logo.png)
The [plan]({attach}diagrams/plan.svg) is attached.
{: .note }

    :::python
    def difference(table):
        return [b - a for a, b in zip(table, table[1:])]

Inline `{filename}not-a-link.md` stays as it is.

*[HTML]: Hyper Text Markup Language

[^1]: Tables of logarithms, mostly.
