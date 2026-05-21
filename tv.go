package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const cacheTTL = 24 * time.Hour

type channel struct {
	title string
	url   string
}

type result struct {
	Title    string `json:"title"`
	URLVideo string `json:"url_video"`
}

type urlQuery struct {
	Queries []query `json:"queries"`
	Size    int     `json:"size"`
}

type query struct {
	Fields []string `json:"fields"`
	Query  string   `json:"query"`
}

var titleReplacer = strings.NewReplacer(
	" lokalzeit", "",
	".", "-",
	" ", "-",
)

func normalizeTitle(title string) string {
	title, _, _ = strings.Cut(title, " Livestream")
	return titleReplacer.Replace(strings.ToLower(title))
}

func channelToURL(channelTitle string, channels []channel) (string, bool) {
	for _, ch := range channels {
		if ch.title == channelTitle {
			return ch.url, true
		}
	}
	return "", false
}

func cacheFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".tv", "mediathekviewweb-response.json"), nil
}

func loadCache() ([]result, bool) {
	path, err := cacheFilePath()
	if err != nil {
		return nil, false
	}
	info, err := os.Stat(path)
	if err != nil || time.Since(info.ModTime()) > cacheTTL {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var results []result
	if err = json.Unmarshal(data, &results); err != nil {
		return nil, false
	}
	return results, true
}

func saveCache(results []result) error {
	path, err := cacheFilePath()
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(results)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func toChannels(results []result) []channel {
	channels := make([]channel, len(results))
	for i, r := range results {
		channels[i] = channel{title: normalizeTitle(r.Title), url: r.URLVideo}
	}
	slices.SortFunc(channels, func(a, b channel) int { return strings.Compare(a.title, b.title) })
	return channels
}

func livestreamURLs(ctx context.Context) ([]channel, error) {
	if results, ok := loadCache(); ok {
		return toChannels(results), nil
	}

	body, err := json.Marshal(urlQuery{
		Queries: []query{
			{Fields: []string{"topic"}, Query: "Livestream"},
			{Fields: []string{"title"}, Query: "Livestream"},
		},
		Size: 100,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		// Found here:
		// https://gist.github.com/Axel-Erfurt/b40584d152e1c2f13259590a135e05f4
		// -> https://59de44955ebd.github.io/tv/index.html
		// -> https://mediathekviewweb.de/api/query?query=
		"https://mediathekviewweb.de/api/query?query="+url.QueryEscape(string(body)),
		nil,
	)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	resp := &struct {
		Result struct {
			Results []result `json:"results"`
		} `json:"result"`
	}{}
	if err = json.Unmarshal(data, resp); err != nil {
		return nil, err
	}

	if err = saveCache(resp.Result.Results); err != nil {
		return nil, err
	}
	return toChannels(resp.Result.Results), nil
}
