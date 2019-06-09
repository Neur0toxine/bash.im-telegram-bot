package main

import (
	"errors"
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
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

var (
	replaceBrRe     = regexp.MustCompile(`(?im)\<[\s+]?br[\s+\/]{0,2}?\>`)
	getQuotesListRe = regexp.MustCompile(`(?im)[.\s\w\W]+?\<article\sclass\="quote\"[.\s\w\W]+?<\/article\>`)
	getQuoteDataRe  = regexp.MustCompile(`(?im)data\-quote\=\"(?P<id>\d+)\"[.\s\w\W]+?quote__header_permalink.+href\=\"(?P<permalink>\/.+\d)\"[.\s\w\W]+?quote__header_date\"\>[.\s\w\W]+?(?P<date>.+)[.\s\w\W]+?quote__body\"\>\s+?(?P<text>.+)[.\s\w\W]+?quote__total.+\>(?P<rating>\d+)`)
)

func getQuotesList(response string, maxItems int) []BashQuote {
	var items []BashQuote
	matches := getQuotesListRe.FindAllString(response, -1)

	if maxItems != 0 && len(matches) > maxItems {
		matches = matches[:maxItems]
	}

	for _, match := range matches {
		id, created, rating, permalink, text, err := getQuoteData(match)

		if err != nil {
			continue
		}

		if id == 0 {
			continue
		}

		items = append(items, BashQuote{
			ID:        id,
			Created:   created,
			Rating:    rating,
			Permalink: permalink,
			Text:      text,
		})
	}

	return items
}

func getQuoteData(response string) (id int, created string, rating string, permalink string, text string, err error) {
	matches := getQuoteDataRe.FindStringSubmatch(response)

	if len(matches) == 0 {
		return 0, "", "", "", "", errors.New("No data found")
	} else {
		matches = matches[1:]
	}

	id, err = strconv.Atoi(matches[0])

	if err != nil {
		return 0, "", "", "", "", err
	}

	created = strings.ReplaceAll(strings.TrimSpace(matches[2]), "  ", " ")
	rating = strings.TrimSpace(matches[4])
	permalink = BASH_URL + matches[1]
	text = html.UnescapeString(replaceBrRe.ReplaceAllString(strings.TrimSpace(matches[3]), "\n"))
	err = nil

	return
}

func GetLatestQuotes() ([]BashQuote, error) {
	return extractQuotes("/", 25)
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
	var (
		quotes []BashQuote
		link   = fmt.Sprintf("%s%s", BASH_URL, url)
	)

	if resp, err := http.Get(link); err == nil {
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			if bodyData, err := ioutil.ReadAll(resp.Body); err == nil {
				body := string(bodyData)
				items := getQuotesList(body, maxItems)

				return items, nil
			} else {
				return quotes, err
			}
		} else {
			return quotes, errors.New("Incorrect status code: " + strconv.Itoa(resp.StatusCode))
		}
	} else {
		return quotes, err
	}
}
