package utils

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// YtResult struct holds yt-dlp command output
type YtResult struct {
	Title    string  `json:"title"`
	Uploader string  `json:"uploader"`
	Duration float32 `json:"duration"`
	Url      string  `json:"webpage_url"`
}

// ProcessQuery determines the type of given query - urls or search.
func ProcessQuery(query string) ([]YtResult, error) {
	t, err := QueryType(query)
	if err != nil {
		return nil, err
	}

	switch t {
	case "search":
		return Get(query)
	case "urls":
		var res []YtResult
		for _, u := range strings.Fields(query) {
			tracks, err := Get(u)
			if err != nil {
				fmt.Println("Failed to fetch", u)
				continue
			}
			res = append(res, tracks...)
		}
		return res, nil
	}
	return nil, errors.New("unable to determine query type")
}

func QueryType(query string) (string, error) {
	allUrls := true
	allWords := true
	for _, u := range strings.Fields(query) {
		if IsUrl(u) {
			allWords = false
		} else {
			allUrls = false
		}
	}
	if allUrls {
		return "urls", nil
	}
	if allWords {
		return "search", nil
	}
	return "", errors.New("either all arguments must be urls or none")
}

func IsUrl(link string) bool {
	parsedURL, err := url.ParseRequestURI(link)
	return err == nil && parsedURL.Scheme != "" && parsedURL.Host != ""
}

func Get(query string) ([]YtResult, error) {
	ytDlp := exec.Command("yt-dlp", "--default-search", "ytsearch",
		"--print", "%(title)s|%(uploader)s|%(duration)s|%(webpage_url)s", query)

	var out, outErr bytes.Buffer
	ytDlp.Stdout = &out
	ytDlp.Stderr = &outErr
	err := ytDlp.Run()
	if err != nil {
		fmt.Println("yt-dlp error:", outErr.String())
		return nil, err
	}

	var tracks []YtResult
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "|")
		if len(parts) < 4 {
			continue
		}
		duration, _ := strconv.ParseFloat(parts[2], 32)
		track := YtResult{
			Title:    parts[0],
			Uploader: parts[1],
			Duration: float32(duration),
			Url:      parts[3],
		}
		tracks = append(tracks, track)
	}

	var wg sync.WaitGroup

	for i, t := range tracks {
		track := t
		index := i
		if track.Title == "" && IsUrl(track.Url) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ytDlp := exec.Command("yt-dlp", "--default-search", "ytsearch",
					"--print", "%(title)s|%(uploader)s|%(duration)s|%(webpage_url)s", track.Url)
				var out bytes.Buffer
				ytDlp.Stdout = &out
				if err := ytDlp.Run(); err != nil {
					fmt.Printf("Error processing %s : %s\n", track.Url, err)
					return
				}
				parts := strings.Split(out.String(), "|")
				if len(parts) < 4 {
					return
				}
				duration, _ := strconv.ParseFloat(parts[2], 32)
				track = YtResult{
					Title:    parts[0],
					Uploader: parts[1],
					Duration: float32(duration),
					Url:      parts[3],
				}
				tracks[index] = track
			}()
		}
	}

	wg.Wait()
	return tracks, nil
}
