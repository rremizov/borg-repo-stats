borg-repo-stats
===============

Show statistics about Borg Backup repository.

Requirements
------------

- Borg Backup v1.1.0 or newer.

Installation
------------

.. code:: sh

    # TODO


Usage
-----

.. code:: sh

    $ borg-repo-stats ~/borg/repository/
    Repository: repository
    Total size: 263 MB
    Archive: 2020-04-21T07:39:01-1.1.9
    Created at: 4 hours from now

    Files by directory (the last archive only):
    a: 41 files, 13 MB
    a/b: 41 files, 13 MB
    a/b/c: 41 files, 13 MB
    a/b/c/d: 41 files, 13 MB
    a/b/c/d/e: 6 files, 509 B
    a/b/c/d/f: 6 files, 6.0 kB
    a/b/c/d/g/h: 6 files, 509 B
    a/b/c/d/i: 5 files, 557 kB
    a/b/c/d/g: 4 files, 2.3 MB
    a/b/c/d/k: 2 files, 72 B
