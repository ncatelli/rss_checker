package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"text/template"

	"github.com/SlyMarbo/rss"
)

const (
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

func loadCachedFeed(feedPath string) (*rss.Feed, error) {
	cachedFeed := &rss.Feed{}

	cachedFileData, err := os.ReadFile(feedPath)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(cachedFileData, cachedFeed); err != nil {
		return nil, err
	}

	return cachedFeed, nil
}

func cacheFeed(cachePath string, feed *rss.Feed) error {
	// mark all items as read prior to caching
	for _, item := range feed.Items {
		item.Read = true
	}
	feed.Unread = 0

	data, err := json.Marshal(feed)
	if err != nil {
		return err
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return err
	}

	return nil
}

func fetchFeed(url string) (resp *http.Response, err error) {
	client := http.DefaultClient
	return client.Get(url)
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
			log.Fatalf("failed to parse feed url with : %s\n", err)
		}

		var newItems []*rss.Item

		feedFile := fmt.Sprintf("%s.json", feedName)
		cacheFile := filepath.Join(cachePath, feedFile)
		feed, err := loadCachedFeed(cacheFile)
		if !errors.Is(err, os.ErrNotExist) {
			err := feed.Update()
			if err != nil {
				log.Fatal(err)
			}
			for _, item := range feed.Items {
				if !item.Read {
					newItems = append(newItems, item)
				}
			}
		} else {
			upstream, err := rss.FetchByFunc(fetchFeed, req.String())

			if err != nil {
				log.Fatalf("invalid upstream url for %s: %s\n", feedName, req.String())
			}

			feed = upstream
			// no new items are append on a new url.
		}

		if cacheFeed(cacheFile, feed) != nil {
			log.Fatalf("failed to cache: %s\n", feedName)
		}

		// setup template
		outputTemplate, err := template.New("output").Parse(formatOutput)
		if err != nil {
			log.Fatalf("failed to prepare output template with: %s\n", err)
		}

		for _, item := range newItems {
			err = outputTemplate.Execute(os.Stdout, item)
			if err != nil {
				log.Fatalf("failed to render output template with: %s\n", err)
			}
		}
	}

}
