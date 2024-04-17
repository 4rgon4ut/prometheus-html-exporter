package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/GusAntoniassi/prometheus-html-exporter/internal/pkg/types"
	"github.com/antchfx/htmlquery"
	log "github.com/sirupsen/logrus"
)

func scrape(config types.ScrapeConfig) (float64, error) {
	log.Debugf("requesting URL '%s'", config.Address)
	body, err := doRequest(config.Address)
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
        return nil, fmt.Errorf("request error: %s", resp.Status)
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
