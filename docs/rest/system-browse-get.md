# rest/system-browse-get.md

Returns a list of directories matching the path given by the optional parameter `current`. The path can use [patterns as described in Go\'s filepath package](https://golang.org/pkg/path/filepath/#Match). A \'\*\' will always be appended to the given path \(e.g. `/tmp/` matches all its subdirectories\). If the option `current` is not given, filesystem root paths are returned.

\`\`\` {.sourceCode .bash} $ curl -H "X-API-Key: yourkey" localhost:8384/rest/system/browse \| json\_pp \[ "/" \]

$ curl -H "X-API-Key: yourkey" localhost:8384/rest/system/browse?current=/var/ \| json\_pp \[ "/var/backups/", "/var/cache/", "/var/lib/", "/var/local/", "/var/lock/", "/var/log/", "/var/mail/", "/var/opt/", "/var/run/", "/var/spool/", "/var/tmp/" \]

$ curl -H "X-API-Key: yourkey" localhost:8384/rest/system/browse?current=/var/\*o \| json\_pp \[ "/var/local/", "/var/lock/", "/var/log/", "/var/opt/", "/var/spool/" \]

\`\`\`

