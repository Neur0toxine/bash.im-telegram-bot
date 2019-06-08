package main

import (
	"errors"
	"io/ioutil"
	"net/http"
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

func GetBashQuote(id int) (BashQuote, error) {
	var quote BashQuote

	if resp, err := http.Get(BASH_URL + "/quote/" + strconv.Itoa(id)); err == nil {
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			if bodyData, err := ioutil.ReadAll(resp.Body); err == nil {
				body := string(bodyData)
				items := getQuotesList(body)

				if len(items) == 1 {
					return items[0], nil
				} else {
					return quote, errors.New("Can't find quote")
				}
			} else {
				return quote, err
			}
		} else {
			return quote, errors.New("Incorrect status code: " + strconv.Itoa(resp.StatusCode))
		}
	} else {
		return quote, err
	}
}

func GetLatestQuotes() ([]BashQuote, error) {
	var quotes []BashQuote

	if resp, err := http.Get(BASH_URL); err == nil {
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			if bodyData, err := ioutil.ReadAll(resp.Body); err == nil {
				body := string(bodyData)
				items := getQuotesList(body)

				if len(items) > 1 {
					return items, nil
				} else {
					return quotes, errors.New("Error while trying to extract quotes")
				}
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

func getQuotesList(response string) []BashQuote {
	re := regexp.MustCompile(`(?im)[.\s\w\W]+?\<article\sclass\="quote\"[.\s\w\W]+?<\/article\>`)
	matches := re.FindAllString(response, -1)
	items := make([]BashQuote, len(matches))

	for index, match := range matches {
		id, created, rating, permalink, text, err := getQuoteData(match)

		if err != nil {
			continue
		}

		items[index] = BashQuote{
			ID:        id,
			Created:   created,
			Rating:    rating,
			Permalink: permalink,
			Text:      text,
		}
	}

	return items
}

func getQuoteData(response string) (id int, created string, rating string, permalink string, text string, err error) {
	re := regexp.MustCompile(`(?im)data\-quote\=\"(?P<id>\d+)\"[.\s\w\W]+?quote__header_permalink.+href\=\"(?P<permalink>\/.+\d)\"[.\s\w\W]+?quote__header_date\"\>[.\s\w\W]+?(?P<date>.+)[.\s\w\W]+?quote__body\"\>\s+?(?P<text>.+)[.\s\w\W]+?quote__total.+\>(?P<rating>\d+)`)
	matches := re.FindStringSubmatch(response)

	if len(matches) == 0 {
		return 0, "", "", "", "", errors.New("No data found")
	} else {
		matches = matches[1:]
	}

	id, err = strconv.Atoi(matches[0])

	if err != nil {
		return 0, "", "", "", "", err
	}

	created = strings.TrimSpace(matches[2])
	rating = strings.TrimSpace(matches[4])
	permalink = BASH_URL + matches[1]
	text = strings.ReplaceAll(strings.TrimSpace(matches[3]), "<br>", "\n")
	err = nil

	return
}
