# rest/db-prio-post.md

Moves the file to the top of the download queue.

\`\`\` {.sourceCode .bash} curl -X POST [http://127.0.0.1:8384/rest/db/prio?folder=default&file=foo/bar](http://127.0.0.1:8384/rest/db/prio?folder=default&file=foo/bar)

\`\`\`

Response contains the same output as `GET /rest/db/need`

