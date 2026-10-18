from pelican import main as pelican_main


def build(c):
    pelican_main(["-s", "pelicanconf.py"])
