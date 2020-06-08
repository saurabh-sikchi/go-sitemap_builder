package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	link "github.com/saurabh-sikchi/go-html_link_parser"
)

type loc struct {
	Value string `xml:"loc"`
}

const xmlns = "http://www.sitemaps.org/schemas/sitemap/0.9"

type urlset struct {
	Urls  []loc  `xml:"url"`
	Xmlns string `xml:"xmlns,attr"`
}

func main() {
	urlFlag := flag.String("url", "", "domain to build sitemap for")
	maxDepth := flag.Int("depth", 3, "the maximum number of links deep to traverse")
	flag.Parse()

	// 1. GET the webpage
	// 2. parse all the links on the webpage using links package
	// 3. build proper urls for the links
	// 4. filter out links to other domains
	// 5. BFS for all the links
	// 6. ouput as XML

	pages := bfs(*urlFlag, *maxDepth)

	x := urlset{
		make([]loc, len(pages)), xmlns,
	}

	for i, p := range pages {
		x.Urls[i] = loc{p}
	}

	enc := xml.NewEncoder(os.Stdout)
	enc.Indent("", "  ")
	fmt.Print(xml.Header)
	if err := enc.Encode(x); err != nil {
		panic(err)
	}
	fmt.Println()

}

func bfs(urlStr string, maxDepth int) []string {
	seen := make(map[string]struct{})
	nq := make(map[string]struct{})
	q := map[string]struct{}{
		urlStr: struct{}{},
	}
	for i := 0; i <= maxDepth; i++ {
		for url := range q {
			if _, found := seen[url]; !found {
				seen[url] = struct{}{}
				for _, page := range get(url) {
					nq[page] = struct{}{}
				}
			}
		}
		q, nq = nq, make(map[string]struct{})
	}

	ret := make([]string, 0, len(seen))
	for url := range seen {
		ret = append(ret, url)
	}
	return ret
}

func get(urlStr string) []string {
	resp, err := http.Get(urlStr)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// /some-path
	// https://domain.com/some-path
	// #fragment
	// mailto:email@domain.com

	reqUrl := resp.Request.URL
	baseUrl := url.URL{
		Scheme: reqUrl.Scheme,
		Host:   reqUrl.Host,
	}
	base := baseUrl.String()
	pages := hrefs(resp.Body, base)
	return filter(pages, withPrefix(base))
}

func hrefs(r io.Reader, base string) (ret []string) {
	links, _ := link.Parse(r)
	for _, l := range links {
		switch {
		case strings.HasPrefix(l.Href, "http"):
			ret = append(ret, l.Href)
		case strings.HasPrefix(l.Href, "/"):
			ret = append(ret, base+l.Href)
		case strings.HasPrefix(l.Href, "#"):
		case strings.HasPrefix(l.Href, "mailto:"):
		default:
			ret = append(ret, base+"/"+l.Href)
		}
	}
	return
}

func filter(links []string, keepFn func(string) bool) (filteredLinks []string) {
	for _, l := range links {
		if keepFn(l) {
			filteredLinks = append(filteredLinks, l)
		}
	}
	return
}

func withPrefix(pfx string) func(string) bool {
	return func(s string) bool {
		return strings.HasPrefix(s, pfx)
	}
}
