package paths

import (
	"io"
	"net/http"
)

func Root(w http.ResponseWriter, req *http.Request) {
	const msg = `
	Thumbs - YouTube Thumbnail Proxy
	https://github.com/javadalmasi/Thumbs

	Routes:
	/stats
	/health`
	io.WriteString(w, msg)
}
