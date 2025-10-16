package paths

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"git.nadeko.net/Fijxu/http3-ytproxy/internal/httpc"
	"git.nadeko.net/Fijxu/http3-ytproxy/internal/utils"
)

func forbiddenChecker(resp *http.Response, w http.ResponseWriter) error {
	if resp.StatusCode == 403 {
		w.WriteHeader(403)
		return fmt.Errorf("forbidden")
	}
	return nil
}

func Vi(w http.ResponseWriter, req *http.Request) {
	const host string = "i.ytimg.com"
	q := req.URL.Query()

	path := req.URL.EscapedPath()

	proxyURL, err := url.Parse("https://" + host + path)
	if err != nil {
		log.Panic(err)
	}

	if strings.HasSuffix(proxyURL.EscapedPath(), "maxres.jpg") {
		proxyURL.Path = utils.GetBestThumbnail(proxyURL.EscapedPath())
	}

	// Pass original query parameters
	proxyURL.RawQuery = q.Encode()

	request, err := http.NewRequest(req.Method, proxyURL.String(), nil)
	if err != nil {
		log.Panic(err)
	}

	request.Header.Set("User-Agent", default_ua)

	resp, err := httpc.Client.Do(request)
	if err != nil {
		log.Panic(err)
	}

	if err := forbiddenChecker(resp, w); err != nil {
		return
	}

	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)

	io.Copy(w, resp.Body)
}
