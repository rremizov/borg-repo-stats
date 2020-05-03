borg-repo-stats
===============

Show statistics about Borg Backup repository.

Features
--------

- Print human-readable text or JSON.

Requirements
------------

- Borg Backup v1.1.0 or newer.

Installation
------------

.. code:: sh

    $ base=https://github.com/rremizov/borg-repo-stats/releases/latest/download/ &&
      curl -L $base/borg-repo-stats.$(uname -s)-$(uname -m) > /tmp/borg-repo-stats &&
      sudo mv /tmp/borg-repo-stats /usr/local/bin/borg-repo-stats &&
      chmod +x /usr/local/bin/borg-repo-stats

Usage
-----

.. code:: sh

    $ borg-repo-stats ~/borg/repository/
    Repository: repository
    Total size: 263 MB
    Archive: 2020-04-21T07:39:01-1.1.9
    Created at: 4 hours from now

    Files count by directory (the last archive only):
    a: 41 files
    a/b: 41 files
    a/b/c: 41 files
    a/b/c/d: 41 files
    a/b/c/d/e: 6 files
    a/b/c/d/f: 6 files
    a/b/c/d/g/h: 6 files
    a/b/c/d/i: 5 files
    a/b/c/d/g: 4 files
    a/b/c/d/k: 2 files

    Size by directory (the last archive only):
    a: 13 MB
    a/b: 13 MB
    a/b/c: 13 MB
    a/b/c/d: 13 MB
    a/b/c/d/e: 509 B
    a/b/c/d/f: 6.0 kB
    a/b/c/d/g/h: 509 B
    a/b/c/d/i: 557 kB
    a/b/c/d/g: 2.3 MB
    a/b/c/d/k: 72 B
