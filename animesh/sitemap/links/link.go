package links

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"golang.org/x/net/html"
)

var verbose bool

type Link struct {
	Href, Text string
}

func Parse(n *html.Node, links *[]Link) {
	switch n.Type {
	case html.ElementNode:
		displayNode("Element", n)
		if n.Data == "a" {
			var l Link
			addLink(n, &l)
			*links = append(*links, l)
			return
		}

	case html.TextNode:
		displayNode("Text", n)
	case html.DocumentNode:
		displayNode("Document", n)
	case html.CommentNode:
		displayNode("Comment", n)
	case html.DoctypeNode:
		displayNode("Doctype", n)
	case html.RawNode:
		displayNode("Raw", n)
	default:
		log.Printf("Unknown node type: %+v\n", n.Type)
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		Parse(c, links)
	}
}

func displayNode(s string, n *html.Node) {
	if verbose {
		fmt.Printf("%s node : %v\n", s, n.Data)
		for _, a := range n.Attr {
			fmt.Printf("\tKey = %v, Val = %v\n", a.Key, a.Val)
		}
	}
}

func addLink(n *html.Node, l *Link) {
	for _, a := range n.Attr {
		if a.Key == "href" {
			l.Href = a.Val
			break
		}
	}
	if n.Type == html.TextNode {
		l.Text = l.Text + n.Data
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		addLink(c, l)
	}
}

func main() {
	filename := flag.String("f", "ex1.html", "HTML file to parse")
	v := flag.Bool("v", false, "Verbose mode")
	flag.Parse()
	verbose = *v

	bytes, err := ioutil.ReadFile(*filename)
	if err != nil {
		log.Fatal(err)
	}

	doc, err := html.Parse(strings.NewReader(string(bytes)))
	if err != nil {
		log.Fatal(err)
	}

	var links []Link

	Parse(doc, &links)

	fmt.Printf("%+v\n", links)
}
