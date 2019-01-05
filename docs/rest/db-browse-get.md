# rest/db-browse-get.md

Returns the directory tree of the global model. Directories are always JSON objects \(map/dictionary\), and files are always arrays of modification time and size. The first integer is the files modification time, and the second integer is the file size.

The call takes one mandatory `folder` parameter and two optional parameters. Optional parameter `levels` defines how deep within the tree we want to dwell down \(0 based, defaults to unlimited depth\) Optional parameter `prefix` defines a prefix within the tree where to start building the structure.

\`\`\` {.sourceCode .bash} $ curl -s [http://localhost:8384/rest/db/browse?folder=default](http://localhost:8384/rest/db/browse?folder=default) \| json\_pp { "directory": { "file": \["2015-04-20T22:20:45+09:00", 130940928\], "subdirectory": { "another file": \["2015-04-20T22:20:45+09:00", 130940928\] } }, "rootfile": \["2015-04-20T22:20:45+09:00", 130940928\] }

$ curl -s [http://localhost:8384/rest/db/browse?folder=default&levels=0](http://localhost:8384/rest/db/browse?folder=default&levels=0) \| json\_pp { "directory": {}, "rootfile": \["2015-04-20T22:20:45+09:00", 130940928\] }

$ curl -s [http://localhost:8384/rest/db/browse?folder=default&levels=1](http://localhost:8384/rest/db/browse?folder=default&levels=1) \| json\_pp { "directory": { "file": \["2015-04-20T22:20:45+09:00", 130940928\], "subdirectory": {} }, "rootfile": \["2015-04-20T22:20:45+09:00", 130940928\] }

$ curl -s [http://localhost:8384/rest/db/browse?folder=default&prefix=directory/subdirectory](http://localhost:8384/rest/db/browse?folder=default&prefix=directory/subdirectory) \| json\_pp { "another file": \["2015-04-20T22:20:45+09:00", 130940928\] }

$ curl -s [http://localhost:8384/rest/db/browse?folder=default&prefix=directory&levels=0](http://localhost:8384/rest/db/browse?folder=default&prefix=directory&levels=0) \| json\_pp { "file": \["2015-04-20T22:20:45+09:00", 130940928\], "subdirectory": {} }

\`\`\`

::: {.note} ::: {.admonition-title} Note :::

This is an expensive call, increasing CPU and RAM usage on the device. Use sparingly. :::

