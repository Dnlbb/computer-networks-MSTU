package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

func main() {
	go startServer()
	time.Sleep(2 * time.Second)
	startClient()
	time.Sleep(1000 * time.Second)
}

func startServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		url := "https://ura.news/msk"

		res, err := http.Get(url)
		if err != nil {
			http.Error(w, "Failed to fetch the URL", http.StatusInternalServerError)
			return
		}
		defer res.Body.Close()

		if res.StatusCode != 200 {
			http.Error(w, "Failed to fetch the page", http.StatusInternalServerError)
			return
		}

		doc, err := html.Parse(res.Body)
		if err != nil {
			http.Error(w, "Failed to parse the document", http.StatusInternalServerError)
			return
		}

		var results []string
		parseHTML(doc, &results)

		for _, result := range results {
			fmt.Fprintln(w, result)
		}

		fmt.Fprintf(w, "<html><body>")
		fmt.Fprintf(w, "<tr><th>ID</th><th>Time</th><th>Title</th></tr>")
		for _, result := range results {
			parts := strings.Split(result, "\n")
			if len(parts) >= 3 {
				id := parts[0][4:]
				time := parts[1][6:]
				title := parts[2][7:]
				fmt.Fprintf(w, "<tr><td>%s</td><td>%s</td><td>%s</td></tr>", id, time, title)
			}
		}
		fmt.Fprintf(w, "</body></html>")
	})

	log.Println("Server is running on port 8082...")
	log.Fatal(http.ListenAndServe(":8082", nil))
}

func startClient() {

	url := "http://localhost:8082"

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Ошибка при отправке запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatalf("Сервер вернул ошибку: %s", resp.Status)
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Ошибка при чтении ответа: %v", err)
		}
		fmt.Print(line)
	}
}

func parseHTML(n *html.Node, results *[]string) {
	if n.Type == html.ElementNode && n.Data == "li" {
		for _, attr := range n.Attr {
			if attr.Key == "class" && strings.Contains(attr.Val, "list-scroll-item") {
				id := getAttributeValue(n, "data-id")
				time := findTime(n)
				title := findTitle(n)
				*results = append(*results, fmt.Sprintf("ID: %s\nTime: %s\nTitle: %s\n", id, time, title))
				break
			}
		}
	}

	for child := n.FirstChild; child != nil; child = child.NextSibling {
		parseHTML(child, results)
	}
}

func getAttributeValue(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func findTime(n *html.Node) string {
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "span" {
			for _, attr := range child.Attr {
				if attr.Key == "class" && attr.Val == "time" {
					if child.FirstChild != nil {
						return child.FirstChild.Data
					}
				}
			}
		}
	}
	return "N/A"
}

func findTitle(n *html.Node) string {
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "a" {
			return getNodeText(child)[10:]
		}
	}
	return "N/A"
}

func getNodeText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var result string
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		result += getNodeText(child)
	}
	return result
}
