Analysis of the Engine
######################

:date: 2023-12-10 18:30
:tags: math, rst
:category: Analysis
:slug: engine-analysis
:summary: The engine, in
    reStructuredText.

.. contents::

Introduction
============

This is *emphasis*, **strong**, and ``inline literal`` text, with
a `link to Python <https://www.python.org/>`_ and a reference to Docutils_.
Inline math: :math:`a^2 + b^2`. A footnote [#]_ and <b>angle brackets</b>.

.. _Docutils: https://docutils.sourceforge.io/

- First item
- Second item with *markup*

  - Nested item

#. Numbered one
#. Numbered two

Details
-------

Here is a literal block::

    def hello():
        return "world"

.. code-block:: python
   :linenos:

   print("highlighted")

.. image:: {static}/images/logo.png
   :alt: The logo

.. figure:: {static}/images/logo.png
   :width: 200px

   The engine logo.

.. note::

   Notes become callouts.

.. warning:: A one-line warning.

.. This is a comment and is dropped.

=====  =====
Input  Output
=====  =====
1      2
3      4
=====  =====

.. graphviz::

   digraph { a -> b }

Term
    Its definition.

----

See also `the engine post <{filename}../engines/difference-engine.md>`_.

.. [#] The footnote text.
