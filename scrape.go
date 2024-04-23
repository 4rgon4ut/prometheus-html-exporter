package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/GusAntoniassi/prometheus-html-exporter/internal/pkg/types"
	"github.com/PuerkitoBio/goquery"
	"github.com/antchfx/htmlquery"
	"github.com/chromedp/chromedp"
	log "github.com/sirupsen/logrus"
)

/*
import (
	"gopkg.in/headzoo/surf.v1"
	"fmt"
)

func main() {
	bow := surf.NewBrowser()
	err := bow.Open("http://golang.org")
	if err != nil {
		panic(err)
	}

	// Outputs: "The Go Programming Language"
	fmt.Println(bow.Title())
}

*/

func scrape(config types.ScrapeConfig) (float64, error) {
	log.Debugf("requesting URL '%s'", config.Address)
	var body io.ReadCloser
	var err error

	if config.Headless {
		body, err = doRequestHeadless(config.Address)
	} else {
		body, err = doRequest(config.Address)
	}
	if err != nil {
		return 0, err
	}

	log.Debugf("scraping value from requested URL with XPath selector '%s'", config.Selector)
	scrapedValue, err := parseSelector(body, config.Selector)

	if err != nil {
		return 0, err
	}

	numberValue, err := normalizeNumericValue(scrapedValue, config.ThousandsSeparator, config.DecimalPointSeparator)
	if err != nil {
		return 0, err
	}

	log.Debugf("scraped value '%0.2f' from URL '%s'", numberValue, config.Address)
	return numberValue, nil
}

func doRequestHeadless(url string) (io.ReadCloser, error) {
	// Create a new context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Run tasks
	// Timeout for running tasks
	ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var htmlContent string

	// Define the tasks to be run, here to navigate to the page and get the HTML
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(10*time.Second), // Wait for JavaScript to execute, adjust timing as necessary
		chromedp.OuterHTML(`html`, &htmlContent, chromedp.ByQuery),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Load the HTML content into goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		log.Fatal("Error loading HTTP response body into goquery document.", err)
	}

	// Use goquery to find the element
	selector := "#ContentPlaceHolder1_divBlocks > div > div"
	doc.Find(selector).Each(func(index int, item *goquery.Selection) {
		linkText := item.Text()
		linkHref, exists := item.Attr("href")
		if exists {
			fmt.Printf("Link found: %s - %s\n", linkText, linkHref)
		} else {
			fmt.Printf("Link found: %s - No href attribute\n", linkText)
		}
	})
	fmt.Println(htmlContent)
	return nil, nil

}

func doRequest(url string) (io.ReadCloser, error) {
    client := &http.Client{
        // Timeout: 10 * time.Second, // Uncomment and adjust as necessary
    }

    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("unable to create request: %s", err)
    }

    // Add headers to mimic a browser request
    req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")
    req.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
    req.Header.Add("Accept-Encoding", "gzip, deflate, br") // Remove 'zstd' if it causes issues
    req.Header.Add("Accept-Language", "en-GB,en;q=0.9")
    req.Header.Add("Cache-Control", "max-age=0")
    req.Header.Add("Sec-Fetch-Dest", "document")
    req.Header.Add("Sec-Fetch-Mode", "navigate")
    req.Header.Add("Sec-Fetch-Site", "cross-site")
    req.Header.Add("Sec-Fetch-User", "?1")
    req.Header.Add("Sec-Gpc", "1")
    req.Header.Add("Upgrade-Insecure-Requests", "1")
    req.Header.Add("Referer", "https://www.google.com/")

	req.Header.Add("Cookie", "ASP.NET_SessionId=0unkgypizygltszqicuwwpln")

    log.Infof("Scraping page %s", url)

    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("unable to request URL %s: %s", url, err)
    }
    defer resp.Body.Close()

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return nil, fmt.Errorf("request error: %s reason: %s", resp.Status, resp.Body)
    }

    return resp.Body, nil
}


func parseSelector(body io.ReadCloser, selector string) (string, error) {
	doc, err := htmlquery.Parse(body)

	if err != nil {
		return "", fmt.Errorf("error loading the response body into XPath nodes. error: %s", err)
	}

	nodes, err := htmlquery.QueryAll(doc, selector)

	if err != nil {
		return "", fmt.Errorf("error querying the XPath expression `%s`. error: %s", selector, err)
	}

	if len(nodes) < 1 {
		return "", fmt.Errorf("no elements returned by the XPath expression `%s`", selector)
	}

	// currently supporting only one attribute. this could change in the future if necessary
	if len(nodes) > 1 {
		log.Warn("more than one element was returned by the XPath expression. only the value of the first element will be exported")
	}

	value := nodes[0].Data

	return value, nil
}

func normalizeNumericValue(value string, thousandsSeparator string, decimalSeparator string) (float64, error) {
	// Replace separators to convert the string into a format accepted by strconv
	value = strings.ReplaceAll(strings.ReplaceAll(value, thousandsSeparator, ""), decimalSeparator, ".")

	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing value %s to a float. error: %s", value, err)
	}

	return floatValue, nil
}
