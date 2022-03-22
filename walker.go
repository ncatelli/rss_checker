package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type ErrEmptyConfFile struct {
	file string
}

func (e *ErrEmptyConfFile) Error() string {
	return fmt.Sprintf("file %s is empty", e.file)
}

type ErrInvalidUrl struct {
	file string
	line string
}

func (e *ErrInvalidUrl) Error() string {
	return fmt.Sprintf("file %s does not contain a valid url: %s", e.file, e.line)
}

func urlFromFirstNonEmptyLine(path string) (*url.URL, error) {
	firstLine := ""

	readFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer readFile.Close()

	scanner := bufio.NewScanner(readFile)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		lineText := scanner.Text()
		if strings.TrimSpace(lineText) == "" {
			continue
		} else {
			firstLine = lineText
			break
		}
	}

	if firstLine == "" {
		return nil, &ErrEmptyConfFile{
			file: path,
		}
	}

	rssUrl, err := url.Parse(firstLine)
	if err != nil {
		return nil, &ErrInvalidUrl{file: path, line: firstLine}
	}

	return rssUrl, nil

}

func WalkAllFilesInConfDir(dir string) (map[string]url.URL, error) {
	feeds := make(map[string]url.URL)

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		} else if !d.Type().IsRegular() {
			return nil
		}

		name := d.Name()
		url, err := urlFromFirstNonEmptyLine(path)
		if err != nil || url == nil {
			return err
		}

		feeds[name] = *url
		return nil
	})

	if err != nil {
		return nil, err
	}

	return feeds, nil
}
