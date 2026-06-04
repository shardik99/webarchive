package main

import (
	"fmt"
	"strings"

	"github.com/derfenix/webarchive/entity"
	"golang.org/x/net/html"
)

func getMetaData(n *html.Node, meta *entity.Meta) {
	if n == nil {
		return
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "title" && c.FirstChild != nil {
			meta.Title = c.FirstChild.Data
		}
		if c.Type == html.ElementNode && c.Data == "meta" {
			attrs := make(map[string]string)
			for _, attr := range c.Attr {
				attrs[attr.Key] = attr.Val
			}

			name, ok := attrs["name"]
			if ok && name == "description" {
				meta.Description = attrs["content"]
			}
		}

		getMetaData(c, meta)
	}
}

func main() {
	htmlStr := `<!DOCTYPE html><html><head><title>Test Title</title></head><body></body></html>`
	node, _ := html.Parse(strings.NewReader(htmlStr))
	
	var fc *html.Node
	for fc = node.FirstChild; fc != nil && fc.Data != "html"; fc = fc.NextSibling {
	}
	
	for fc = fc.FirstChild; fc != nil && fc.Data != "head"; fc = fc.NextSibling {
	}
	
	if fc == nil {
		fmt.Println("failed to find head")
		return
	}
	
	meta := &entity.Meta{}
	getMetaData(fc, meta)
	fmt.Printf("Meta: %+v\n", meta)
}
