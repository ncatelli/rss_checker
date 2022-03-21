package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/gorilla/feeds"
)

const (
	cacheDir            string = ".rss_checker/cache/"
	cachedFeedFmtString string = "%s%s.xml"
)

func getEnvOr(key, defaultVal string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return defaultVal
	} else {
		return val
	}
}

type ErrInvalidFeedResponse struct {
	feed     string
	respCode int
}

func (e *ErrInvalidFeedResponse) Error() string {
	return fmt.Sprintf("received invalid responce code (%d) from feed", e.respCode)
}

func fetchFeed(req *url.URL) (*feeds.RssFeedXml, error) {
	feed := &feeds.RssFeedXml{}

	res, err := http.Get(req.String())
	if err != nil {
		log.Print(err)
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if res.StatusCode > 299 {
		return nil, &ErrInvalidFeedResponse{feed: req.String(), respCode: res.StatusCode}
	}

	if err = xml.Unmarshal(body, feed); err != nil {
		return nil, err
	}

	return feed, nil
}

type Feed struct {
	name string
	data *feeds.RssFeedXml
}

func compareCache(cacheDir string, feed Feed, comparator func(cacheditem, upstreamItem *feeds.RssItem) bool) ([]*feeds.RssItem, error) {
	items := make([]*feeds.RssItem, 0, 10)
	cachedFeed := &feeds.RssFeedXml{}

	fileName := fmt.Sprintf(cachedFeedFmtString, cacheDir, feed.name)
	data, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	err = xml.Unmarshal(data, cachedFeed)
	if err != nil {
		return nil, err
	}

	for _, upstreamItem := range feed.data.Channel.Items {
		found := false
		for _, cachedItem := range cachedFeed.Channel.Items {
			if comparator(cachedItem, upstreamItem) {
				found = true
				break
			}
		}

		if !found {
			items = append(items, upstreamItem)
		}
	}

	return items, nil
}

func cacheFeed(cacheDir string, feed Feed) error {
	fileName := fmt.Sprintf(cachedFeedFmtString, cacheDir, feed.name)
	data, err := xml.Marshal(feed.data)
	if err != nil {
		return err
	}

	if err := os.WriteFile(fileName, data, 0644); err != nil {
		return err
	}

	return nil
}

func main() {
	log.Println("starting...")
	feedConfs, err := WalkAllFilesInConfDir("conf/")
	if err != nil {
		log.Fatal("failed to fetch feeds")
	}

	for feedName, feedUrl := range feedConfs {
		req, err := url.Parse(feedUrl.String())
		if err != nil {
			log.Printf("failed to fetch feed: %s", feedName)
			continue
		}

		rawFeed, err := fetchFeed(req)
		if err != nil {
			fmt.Printf("failed to unmarshal feed: %s", feedName)
			continue
		}

		feed := Feed{
			name: feedName,
			data: rawFeed,
		}

		var newItems []*feeds.RssItem
		cacheFile := fmt.Sprintf(cachedFeedFmtString, cacheDir, feed.name)
		if stat, err := os.Stat(cacheFile); !errors.Is(err, os.ErrNotExist) {
			newItems, err = compareCache(cacheDir, feed, func(cacheditem, upstreamItem *feeds.RssItem) bool {
				return cacheditem.Link == upstreamItem.Link
			})
			if err != nil {
				fmt.Printf("failed to compare new feed to cache: %s", feedName)
				continue
			}
		} else if err == nil && stat.IsDir() {
			fmt.Printf("cache is dir: %s", feedName)
			continue
		} else {
			for _, item := range feed.data.Channel.Items {
				newItems = append(newItems, item)
			}
		}

		err = cacheFeed(cacheDir, feed)
		if err != nil {
			fmt.Printf("failed to cache: %s", feedName)
			continue
		}

		for _, i := range newItems {
			fmt.Printf("title: %s\nlink: %s\ndescription: %s\n\n", i.Title, i.Link, i.Description)
		}
	}

}
