package main

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	uurl "net/url"
	"os"
	"sort"
	"strings"

	. "../links"

	"golang.org/x/net/html"
)

var (
	schemes  = [...]string{"https://", "http://"}
	mLinks   = make(map[string]Link, 0)
	maxDepth = -1
	rootURL  = ""
)

func hasScheme(s, withOption string) bool {
	for _, p := range schemes {
		if strings.HasPrefix(s, p+withOption) {
			return true
		}
	}
	return false
}

func loadPage(u string) ([]byte, error) {
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func filterLinks(links []Link, f func(Link) bool) []Link {
	res := make([]Link, 0)
	for _, l := range links {
		if f(l) {
			res = append(res, l)
		}
	}
	return res
}

func mapLinks(links []Link, f func(Link) Link) []Link {
	res := make([]Link, 0)
	for _, l := range links {
		res = append(res, f(l))
	}
	return res
}

func hasDomain(s, domain string) bool {
	return len(s) > 1 && (string(s[0]) == "/" || hasScheme(s, domain+"/"))
}

func purgeTwins(links []Link) []Link {
	uniqLinks := make([]Link, 0)
	sort.Slice(links, func(i1, i2 int) bool {
		return links[i1].Href < links[i2].Href
	})
	for _, l := range links {
		i := sort.Search(len(uniqLinks), func(i int) bool {
			return l.Href == uniqLinks[i].Href
		})
		if i >= len(uniqLinks) {
			uniqLinks = append(uniqLinks, l)
		}
	}
	return uniqLinks
}

func parsePage(u, domain string) ([]Link, error) {
	body, err := loadPage(u)
	if err != nil {
		return nil, err
	}

	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	
	var links []Link
	Parse(doc, &links)

	filteredLinks := filterLinks(links, func(l Link) bool {
		ref := strings.TrimSpace(l.Href)
		return ref != "" && hasDomain(ref, domain)
	})
	
	normalizedLinks := mapLinks(filteredLinks, func(l Link) Link {
		if string(l.Href[0]) == "/" {
			return Link{Href: rootURL + l.Href, Text: l.Text}
		}
		return l
	})

	uniqLinks := purgeTwins(normalizedLinks)

	return uniqLinks, nil
}

func findAll(links []Link, domain string, depth int) error {
	for _, link := range links {
		if _, found := mLinks[link.Href]; found {
			continue
		}

		mLinks[link.Href] = link
		
		if maxDepth == -1 || depth < maxDepth {
			newLinks, err := parsePage(link.Href, domain)
			if err != nil {
				return err
			}
			findAll(newLinks, domain, depth+1)
		}
	}
	return nil
}

type url struct {
	Loc string `xml:"loc"`
}

func writeXml(filename string) error {
	urls := make([]url, 0)
	for _, l := range mLinks {
		urls = append(urls, url{Loc: l.Href})
	}
	sort.Slice(urls, func(i1, i2 int) bool {
		return urls[i1].Loc < urls[i2].Loc
	})

	output, err := xml.MarshalIndent(&urls, "  ", "   ")
	if err != nil {
		return err
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Write([]byte(xml.Header))
	f.Write([]byte("<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">"))
	f.Write(output)
	f.Write([]byte("</urlset>"))
	return nil
}

// Write the urls to stdout
func writeBuf() {
	fmt.Printf("Number of links : %d\n", len(mLinks))
	urls := make([]string, 0)
	for _, l := range mLinks {
		urls = append(urls, l.Href)
	}
	sort.Strings(urls)
	w := bufio.NewWriter(os.Stdout)
	fmt.Fprint(w, urls)
	w.Flush()
}

func extractDomain(url string) (string, error) {
	u, err := uurl.Parse(url)
	if err != nil {
		return "", err
	}
	h := strings.Split(u.Hostname(), ".")
	domain := h[len(h)-2] + "." + h[len(h)-1]
	return domain, nil
}

func findRootUrl(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	return resp.Request.URL.String(), nil
}

func main() {
	url := flag.String("domain", "https://animesh-kumar-singh.herokuapp.com", "domain to parse")
	outfile := flag.String("outfile", "", "XML output file (if empty, output is written to stdout)")
	depth := flag.Int("depth", -1, "max depth number of links to follow from root page")
	flag.Parse()

	domain, err := extractDomain(*url)
	if err != nil {
		panic(err)
	}

	rootURL, err = findRootUrl(*url)
	if err != nil {
		panic(err)
	}

	maxDepth = *depth

	links := []Link{{Href: *url, Text: "Root link"}}

	err = findAll(links, domain, 0)
	if err != nil {
		panic(err)
	}

	if strings.TrimSpace(*outfile) == "" {
		writeBuf()
	} else {
		writeXml(*outfile)
	}
}
