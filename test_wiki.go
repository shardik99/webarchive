package main

import (
	"fmt"
	"net/http"

	"golang.org/x/net/html"
)

type Meta struct {
	Title       string
	Description string
}

func getMetaData(n *html.Node, meta *Meta) {
	if n == nil {
		return
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "title" {
			if c.FirstChild != nil {
				meta.Title = c.FirstChild.Data
			}
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
	req, _ := http.NewRequest("GET", "https://en.wikipedia.org/wiki/March_18_Massacre", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 WebArchive/1.0")
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	htmlNode, err := html.Parse(resp.Body)
	if err != nil {
		panic(err)
	}

	var fc *html.Node
	for fc = htmlNode.FirstChild; fc != nil; fc = fc.NextSibling {
		if fc.Type == html.ElementNode && fc.Data == "html" {
			break
		}
	}

	if fc == nil {
		fmt.Println("failed to find html tag")
		return
	}

	for fc = fc.FirstChild; fc != nil; fc = fc.NextSibling {
		if fc.Type == html.ElementNode && fc.Data == "head" {
			break
		}
	}

	if fc == nil {
		fmt.Println("failed to find head tag")
		return
	}

	meta := &Meta{}
	getMetaData(fc, meta)
	fmt.Printf("Extracted Meta: %+v\n", meta)
}
