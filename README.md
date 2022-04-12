# rss_checker
<!-- TOC -->

- [rss_checker](#rss_checker)
	- [General](#general)
		- [Examples](#examples)
	- [Dependencies](#dependencies)
		- [Dev](#dev)
	- [Building](#building)
		- [Docker](#docker)
	- [Testing](#testing)
	- [Configuration](#configuration)
		- [Command-Line Options](#command-line-options)
		- [Configuration files](#configuration-files)
		- [Automation through Github Actions](#automation-through-github-actions)

<!-- /TOC -->
## General
This tool provides a simple to use RSS feed checker that makes use of the flexibility of github actions to provide flexibility.

Examples of simple configurations can be seen below

### Examples
- [Daily feed check](./examples/actions/check_feeds.yaml)
- [Daily feed check that posts to discord](./examples/actions/check_feeds_discord.yaml)

## Dependencies
### Dev
- docker-compose

## Building
### Docker
The tool can be built and run entirely via docker using the following command.

```sh
$> docker build -t ncatelli/mockserver .
```

## Testing
An integration test environment has been provided in the enclosed docker-compose.yaml file and the checker can be access by running the following commands.

```sh
$> docker-compose up -d
$> docker-compose run --entrypoint sh checker
```

## Configuration
### Command-Line Options
```
  -format string
        a formatting string for the resulting output data (default "{{ .Link }}\n")
  -help
        print help information
```
- cache-path: `string`  The relative directory path to store all cache files.  [".rss_checker/cache"]
- conf-path:  `string`  The relative directory source feed configuration files. ["./conf"]
- format:     `url.URL` A go-template formatting string for the resulting output data. Context contains a [Feed](https://pkg.go.dev/github.com/SlyMarbo/rss#Feed). ["{{ .Link }}\n"]

### Configuration files
Feeds are configured by placing single-line, uniquely-named files in the conf-path whose contents are _ONLY_ a url for a given sites feed.

```sh
$> cat conf/example.com
https://example.com/rss.xml
$>
```

Each filename serves as a unique cache key and, on invocation, the result of the feed is compared against the local cache to identify any items.

### Automation through Github Actions
Two [examples](#examples) have been provided that demonstrate how to configure this tool to run at a daily scheduled interval via github actions.

To provide for consistent caching, The examples make use of the [actions build cache](https://github.com/actions/cache) is configured to provide a sequential daily cache against previous runs.

The included examples only cover shipping logs to discord. However alternatives can easily be used, and formatted using the built in `-format` flag and github's action's library.
