package main

import (
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/mgutz/logxi/v1"
)

var (
	errNoPlayerFound = errors.New("no HTML5 player instance found")
)

// ZingAlbumItem represents each item in ZingAlbum.
type ZingAlbumItem struct {
	Title       string `xml:"title"`
	Artist      string `xml:"performer"`
	ItemURL     string `xml:"link"`
	DownloadURL string `xml:"source"`
	LyricURL    string `xml:"lyric"`
}

// ZingAlbum represents a Zing MP3 player source.
type ZingAlbum struct {
	XMLName xml.Name        `xml:"data"`
	Items   []ZingAlbumItem `xml:"item"`
}

// ParseAlbumData parses a zing MP3 URL and returns a ZingAlbum associated with the current player on the page.
func ParseAlbumData(zingURL string) (*ZingAlbum, error) {
	doc, err := goquery.NewDocument(zingURL)
	if err != nil {
		return nil, err
	}

	dataXMLURL, found := doc.Find("div#html5player").Attr("data-xml")
	if found == false {
		return nil, errNoPlayerFound
	}

	response, err := http.Get(dataXMLURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	album := &ZingAlbum{}
	err = xml.NewDecoder(response.Body).Decode(album)
	if err != nil {
		return nil, err
	}

	return album, nil
}

// DownloadAlbumItem fetches the song from DownloadURL and returns an os.File which represents the file on-disk.
func DownloadAlbumItem(item *ZingAlbumItem) (*os.File, error) {
	response, err := http.Get(item.DownloadURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	filename := fmt.Sprintf("%s - %s.mp3",
		strings.TrimSpace(item.Artist),
		strings.TrimSpace(item.Title),
	)
	fd, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	io.Copy(fd, response.Body)
	return fd, nil
}

// DownloadAlbum initializes
func DownloadAlbum(zingURL string) error {
	album, err := ParseAlbumData(zingURL)
	if err != nil {
		return err
	}

	log.Debug("Found items",
		"item_count", len(album.Items),
		"album_url", zingURL,
	)
	for _, item := range album.Items {
		log.Info("Processing item",
			"artist", item.Artist,
			"title", item.Title,
			"download_url", item.DownloadURL,
		)
		fd, _ := DownloadAlbumItem(&item)
	}

	return nil
}

func main() {
	var (
		zingURL = flag.String("url", "", "Zing MP3 URL to be parsed")
	)
	flag.Parse()

	DownloadAlbum(*zingURL)
}
