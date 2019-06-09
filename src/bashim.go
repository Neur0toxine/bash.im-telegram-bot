package main

import (
	"errors"
	"fmt"
	"github.com/gocolly/colly"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const BASH_URL = "https://bash.im"

type BashQuote struct {
	ID        int
	Created   string
	Rating    string
	Permalink string
	Text      string
}

var replaceBrRe = regexp.MustCompile(`(?im)\<[\s+]?br[\s+\/]{0,2}?\>`)

func GetLatestQuotes() ([]BashQuote, error) {
	return extractQuotes("/", 25)
}

func GetLatestAbyssQuotes() ([]BashQuote, error) {
	return extractQuotes("/abyss/", 25)
}

func GetQuote(id int) (BashQuote, error) {
	quotes, err := extractQuotes(fmt.Sprintf("/quote/%d", id), 1)

	if err != nil {
		return BashQuote{}, err
	} else if len(quotes) == 0 {
		return BashQuote{}, errors.New("Error while trying to extract quote")
	} else {
		return quotes[0], nil
	}
}

func SearchQuotes(search string, maxResults int) ([]BashQuote, error) {
	return extractQuotes(fmt.Sprintf("/search?text=%s", url.QueryEscape(search)), maxResults)
}

func extractQuotes(url string, maxItems int) ([]BashQuote, error) {
	var quotes []BashQuote

	c := colly.NewCollector()

	c.OnHTML("article.quote", func(e *colly.HTMLElement) {
		if len(quotes) == maxItems {
			return
		}

		id, err := strconv.Atoi(strings.TrimSpace(e.Attr("data-quote")))

		if err == nil {
			created, err := e.DOM.Find(".quote__header_date").Html()
			rating, err := e.DOM.Find(".quote__total").Html()
			permalink, err := e.DOM.Find(".quote__header_permalink").Html()
			text, err := e.DOM.Find(".quote__body").Html()

			if err == nil {
				created = strings.TrimSpace(strings.ReplaceAll(created, "  ", " "))
				rating = strings.TrimSpace(rating)
				text = replaceBrRe.ReplaceAllString(html.UnescapeString(strings.TrimSpace(text)), "\n")
				text = strings.ReplaceAll(text, "`", "\\`")
				text = strings.ReplaceAll(text, "*", "\\*")
				text = strings.ReplaceAll(text, "_", "\\_")

				if len(permalink) > 0 && permalink[0] != '#' {
					permalink = fmt.Sprintf("%s%s", BASH_URL, strings.TrimSpace(permalink))
				} else {
					permalink = ""
				}

				quotes = append(quotes, BashQuote{
					ID:        id,
					Created:   created,
					Rating:    rating,
					Permalink: permalink,
					Text:      text,
				})
			}
		} else {
			return
		}
	})

	if err := c.Visit(fmt.Sprintf("%s%s", BASH_URL, url)); err != nil {
		return quotes, err
	} else {
		return quotes, nil
	}
}
