package main

import (
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"text/template"

	"github.com/gorilla/feeds"
)

const (
	cachedFeedFmtString     string = "%s/%s.xml"
	defaultOutputFormatting string = "{{ .Link }}\n"
)

var (
	confPath     string
	cachePath    string
	formatOutput string
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

func printHelp() {
	fmt.Println("Usage: rss_checker [OPTIONS]...")
	fmt.Printf("A cli RSS feed checker.\n\n")
	flag.PrintDefaults()
}

func main() {
	help := flag.Bool("help", false, "print help information")
	flag.StringVar(&confPath, "conf-path", getEnvOr("RSS_CHECKER_CONF_PATH", "conf"), "the directory path to source conf files")
	flag.StringVar(&cachePath, "cache-path", getEnvOr("RSS_CHECKER_CACHE_PATH", ".rss_checker/cache"), "the directory path to store all cache files")
	flag.StringVar(&formatOutput, "format", getEnvOr("RSS_CHECKER_OUTPUT_FORMAT", defaultOutputFormatting), "a formatting string for the resulting output data")
	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	feedConfs, err := WalkAllFilesInConfDir(filepath.Clean(confPath))
	if err != nil {
		log.Fatal("failed to fetch feeds")
	}

	for feedName, feedUrl := range feedConfs {
		req, err := url.Parse(feedUrl.String())
		if err != nil {
			log.Fatalf("failed to fetch feed: %s", feedName)
			continue
		}

		rawFeed, err := fetchFeed(req)
		if err != nil {
			log.Fatalf("failed to unmarshal feed: %s", feedName)
			continue
		}

		feed := Feed{
			name: feedName,
			data: rawFeed,
		}

		var newItems []*feeds.RssItem

		feedFile := fmt.Sprintf("%s.xml", feed.name)
		cacheFile := filepath.Join(cachePath, feedFile)
		if stat, err := os.Stat(cacheFile); !errors.Is(err, os.ErrNotExist) {
			newItems, err = compareCache(cachePath, feed, func(cacheditem, upstreamItem *feeds.RssItem) bool {
				return cacheditem.Link == upstreamItem.Link
			})
			if err != nil {
				log.Fatalf("failed to compare new feed to cache: %s", feedName)
			}
		} else if err == nil && stat.IsDir() {
			fmt.Printf("cache is dir: %s\n", feedName)
			continue
		}

		err = cacheFeed(cachePath, feed)
		if err != nil {
			log.Fatalf("failed to cache: %s\n", feedName)
		}

		// setup template
		outputTemplate, err := template.New("output").Parse(formatOutput)
		if err != nil {
			log.Fatal(err)
		}

		for _, item := range newItems {
			err = outputTemplate.Execute(os.Stdout, item)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

}
