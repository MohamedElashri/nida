#!/usr/bin/env python
# -*- coding: utf-8 -*- #
import os

AUTHOR = 'Ada Lovelace'
SITENAME = "Analytical Notes"
SITESUBTITLE = 'Notes on ' \
    'engines'
SITEURL = ''

PATH = 'content'
TIMEZONE = 'Europe/Paris'
DEFAULT_LANG = 'en'

ARTICLE_URL = '{date:%Y}/{date:%m}/{slug}.html'
ARTICLE_SAVE_AS = '{date:%Y}/{date:%m}/{slug}.html'
TAG_URL = 'tag/{slug}/'
TAG_SAVE_AS = 'tag/{slug}/index.html'

STATIC_PATHS = ['images', 'extra/robots.txt', 'extra/CNAME']
EXTRA_PATH_METADATA = {
    'extra/robots.txt': {'path': 'robots.txt'},
    'extra/CNAME': {'path': 'CNAME'},
}

# Feed generation is usually not desired when developing
FEED_ALL_ATOM = None
CATEGORY_FEED_ATOM = None
TRANSLATION_FEED_ATOM = None
AUTHOR_FEED_ATOM = None
AUTHOR_FEED_RSS = None

LINKS = (('Pelican', 'https://getpelican.com/'),
         ('Python.org', 'https://www.python.org/'),)
SOCIAL = (('github', 'https://github.com/ada'),)
MENUITEMS = [("Archives", "/archives.html")]

DEFAULT_PAGINATION = 10

PLUGIN_PATHS = ['plugins']
PLUGINS = ['sitemap', 'neighbors', 'render_math']

MARKDOWN = {
    'extension_configs': {
        'markdown.extensions.codehilite': {'css_class': 'highlight'},
        'markdown.extensions.extra': {},
        'markdown.extensions.toc': {'permalink': True},
    },
    'output_format': 'html5',
}

THEME = 'themes/quiet'
GITHUB_URL = 'https://github.com/ada'
DISQUS_SITENAME = os.environ.get('DISQUS', '')

if os.environ.get('DEBUG'):
    SITENAME = 'Debug build'

RELATIVE_URLS = True
