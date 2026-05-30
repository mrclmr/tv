package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNormalizeTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"ARD Livestream", "ard"},
		{"ZDF Livestream", "zdf"},
		{"3sat Livestream", "3sat"},
		{"arte Livestream", "arte"},
		{"HR Fernsehen Livestream", "hr-fernsehen"},
		{"SR Fernsehen Livestream", "sr-fernsehen"},
		{"N-TV Livestream", "n-tv"},
		{"tagesschau24 Livestream", "tagesschau24"},
		{"WDR Lokalzeit Aachen Livestream", "wdr-aachen"},
		{"No suffix", "no-suffix"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := normalizeTitle(tt.input); got != tt.want {
				t.Errorf("normalizeTitle(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestChannelToURL(t *testing.T) {
	channels := []channel{
		{title: "ard", url: "https://example.com/ard.m3u8"},
		{title: "zdf", url: "https://example.com/zdf.m3u8"},
	}
	tests := []struct {
		title string
		want  string
		found bool
	}{
		{"ard", "https://example.com/ard.m3u8", true},
		{"zdf", "https://example.com/zdf.m3u8", true},
		{"unknown", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got, found := channelToURL(tt.title, channels)
			if found != tt.found {
				t.Errorf("channelToURL(%q) found = %v, want %v", tt.title, found, tt.found)
			}
			if got != tt.want {
				t.Errorf("channelToURL(%q) url = %q, want %q", tt.title, got, tt.want)
			}
		})
	}
}

func TestToChannels(t *testing.T) {
	results := []result{
		{Title: "ZDF Livestream", URLVideo: "https://example.com/zdf.m3u8"},
		{Title: "ARD Livestream", URLVideo: "https://example.com/ard.m3u8"},
		{Title: "WDR Lokalzeit Aachen Livestream", URLVideo: "https://example.com/wdr-aachen.m3u8"},
	}
	channels := toChannels(results)

	want := []channel{
		{title: "ard", url: "https://example.com/ard.m3u8"},
		{title: "wdr-aachen", url: "https://example.com/wdr-aachen.m3u8"},
		{title: "zdf", url: "https://example.com/zdf.m3u8"},
	}
	if len(channels) != len(want) {
		t.Fatalf("toChannels() len = %d, want %d", len(channels), len(want))
	}
	for i, ch := range channels {
		if ch != want[i] {
			t.Errorf("toChannels()[%d] = %+v, want %+v", i, ch, want[i])
		}
	}
}

func TestResolveAlias(t *testing.T) {
	channels := []channel{
		{title: "arte-de", url: "https://example.com/arte.m3u8"},
		{title: "zdf", url: "https://example.com/zdf.m3u8"},
	}
	tests := []struct {
		input string
		want  string
	}{
		{"arte", "arte-de"},
		{"deutsche-welle", "deutsche-welle"},
		{"zdf", "zdf"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := resolveAlias(tt.input, channels); got != tt.want {
				t.Errorf("resolveAlias(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalAudioTrack(t *testing.T) {
	tests := []struct {
		name     string
		manifest string
		want     int
	}{
		{
			name: "normal audio first",
			manifest: `#EXTM3U
#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="audio",LANGUAGE="de",NAME="Deutsch",DEFAULT=YES,AUTOSELECT=YES,URI="audio1.m3u8"
#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="audio",LANGUAGE="de",NAME="Audiodeskription",CHARACTERISTICS="public.accessibility.describes-video",DEFAULT=NO,AUTOSELECT=YES,URI="audio2.m3u8"`,
			want: 1,
		},
		{
			name: "audio description first",
			manifest: `#EXTM3U
#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="audio",LANGUAGE="de",NAME="Audiodeskription",CHARACTERISTICS="public.accessibility.describes-video",DEFAULT=YES,AUTOSELECT=YES,URI="audio1.m3u8"
#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="audio",LANGUAGE="de",NAME="Deutsch",DEFAULT=NO,AUTOSELECT=YES,URI="audio2.m3u8"`,
			want: 2,
		},
		{
			name: "normal audio amid multiple tracks",
			manifest: `#EXTM3U
#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="audio",LANGUAGE="de",NAME="Audiodeskription",CHARACTERISTICS="public.accessibility.describes-video",DEFAULT=YES,URI="audio1.m3u8"
#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="audio",LANGUAGE="de",NAME="Klare Sprache",DEFAULT=NO,URI="audio2.m3u8"
#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="audio",LANGUAGE="de",NAME="Deutsch",DEFAULT=NO,URI="audio3.m3u8"`,
			want: 2,
		},
		{
			name: "only audio description track",
			manifest: `#EXTM3U
#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="audio",LANGUAGE="de",NAME="Audiodeskription",CHARACTERISTICS="public.accessibility.describes-video",DEFAULT=YES,URI="audio1.m3u8"`,
			want: 0,
		},
		{
			name: "no audio tracks",
			manifest: `#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=1000000
stream.m3u8`,
			want: 0,
		},
		{
			name: "multiple groups only counts first group",
			manifest: `#EXTM3U
#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="group1",LANGUAGE="de",NAME="Deutsch",DEFAULT=YES,URI="g1audio1.m3u8"
#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="group1",LANGUAGE="de",NAME="Audiodeskription",CHARACTERISTICS="public.accessibility.describes-video",DEFAULT=NO,URI="g1audio2.m3u8"
#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="group2",LANGUAGE="de",NAME="Deutsch",DEFAULT=YES,URI="g2audio1.m3u8"
#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="group2",LANGUAGE="de",NAME="Audiodeskription",CHARACTERISTICS="public.accessibility.describes-video",DEFAULT=NO,URI="g2audio2.m3u8"`,
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tt.manifest))
			}))
			defer srv.Close()
			got := normalAudioTrack(context.Background(), srv.URL)
			if got != tt.want {
				t.Errorf("normalAudioTrack() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNormalAudioTrackUnreachable(t *testing.T) {
	got := normalAudioTrack(context.Background(), "http://127.0.0.1:0/invalid.m3u8")
	if got != 0 {
		t.Errorf("normalAudioTrack() = %d, want 0 for unreachable URL", got)
	}
}

func TestCacheRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	results := []result{
		{Title: "ARD Livestream", URLVideo: "https://example.com/ard.m3u8"},
		{Title: "ZDF Livestream", URLVideo: "https://example.com/zdf.m3u8"},
	}

	if err := saveCache(results); err != nil {
		t.Fatalf("saveCache() error = %v", err)
	}

	got, ok := loadCache()
	if !ok {
		t.Fatal("loadCache() returned no data")
	}
	if len(got) != len(results) {
		t.Fatalf("loadCache() len = %d, want %d", len(got), len(results))
	}
	for i, r := range got {
		if r != results[i] {
			t.Errorf("loadCache()[%d] = %+v, want %+v", i, r, results[i])
		}
	}
}

func TestCacheExpired(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	if err := saveCache([]result{{Title: "ARD Livestream", URLVideo: "https://example.com/ard.m3u8"}}); err != nil {
		t.Fatalf("saveCache() error = %v", err)
	}

	path := filepath.Join(dir, ".tv", "mediathekviewweb-response.json")
	expired := time.Now().Add(-(cacheTTL + time.Second))
	if err := os.Chtimes(path, expired, expired); err != nil {
		t.Fatalf("Chtimes() error = %v", err)
	}

	if _, ok := loadCache(); ok {
		t.Error("loadCache() returned data for expired cache, want miss")
	}
}
