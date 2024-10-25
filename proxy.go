package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)


const proxyBaseURL = "http://185.102.139.169:7081"


func modifyLinksWithGoQuery(htmlContent string, baseURL *url.URL) (string, error) {
  doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
  if err != nil {
    return "", err
  }

  toAbsoluteURL := func(link string) string {
    u, err := url.Parse(link)
    if err != nil {
      return link
    }
    return baseURL.ResolveReference(u).String()
  }

  doc.Find("a").Each(func(i int, s *goquery.Selection) {
    if href, exists := s.Attr("href"); exists {
      absoluteHref := toAbsoluteURL(href)
      newHref := fmt.Sprintf("%s/?url=%s", proxyBaseURL, absoluteHref)
      s.SetAttr("href", newHref)
    }
  })

 
  doc.Find("img").Each(func(i int, s *goquery.Selection) {
    if src, exists := s.Attr("src"); exists {
      absoluteSrc := toAbsoluteURL(src)
      newSrc := fmt.Sprintf("%s/?url=%s", proxyBaseURL, absoluteSrc)
      s.SetAttr("src", newSrc)
    }
  })

  doc.Find("link[rel='stylesheet']").Each(func(i int, s *goquery.Selection) {
    if href, exists := s.Attr("href"); exists {
      absoluteHref := toAbsoluteURL(href)
      newHref := fmt.Sprintf("%s/?url=%s", proxyBaseURL, absoluteHref)
      s.SetAttr("href", newHref)
    }
  })

  doc.Find("script").Each(func(i int, s *goquery.Selection) {
    if src, exists := s.Attr("src"); exists {
      absoluteSrc := toAbsoluteURL(src)
      newSrc := fmt.Sprintf("%s/?url=%s", proxyBaseURL, absoluteSrc)
      s.SetAttr("src", newSrc)
    }
  })

  html, err := doc.Html()
  if err != nil {
    return "", err
  }

  return html, nil
}

func handleProxy(w http.ResponseWriter, r *http.Request) {
  
  targetURL := r.URL.Query().Get("url")
  if targetURL == "" {
    http.Error(w, "URL parameter is missing", http.StatusBadRequest)
    return
  }

  parsedURL, err := url.Parse(targetURL)
  if err != nil || !parsedURL.IsAbs() {
    http.Error(w, "Invalid URL", http.StatusBadRequest)
    return
  }

  
  resp, err := http.Get(targetURL)
  if err != nil {
    http.Error(w, "Error making request to the target URL", http.StatusBadGateway)
    return
  }
  defer resp.Body.Close()

  body, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    http.Error(w, "Error reading response from the target URL", http.StatusInternalServerError)
    return
  }

  contentType := resp.Header.Get("Content-Type")
  if strings.Contains(contentType, "text/html") {
    modifiedBody, err := modifyLinksWithGoQuery(string(body), parsedURL)
    if err != nil {
      http.Error(w, "Error processing HTML", http.StatusInternalServerError)
      return
    }
    body = []byte(modifiedBody)
  }

  for key, values := range resp.Header {
    for _, value := range values {
      w.Header().Add(key, value)
    }
  }
  w.WriteHeader(resp.StatusCode)
  w.Write(body)
}

func main() {
  
  http.HandleFunc("/", handleProxy)

  log.Println("7081")
  log.Fatal(http.ListenAndServe(":7081", nil))
}
