package main

import (
	"encoding/xml"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"fmt"

	"github.com/PuerkitoBio/goquery"
	"io"
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

// GetAlbum parses a zing MP3 URL and returns a ZingAlbum associated with the current player on the page.
func GetAlbum(zingURL string) (*ZingAlbum, error) {
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

func DownloadAlbumItem(item *ZingAlbumItem) error {
	response, err := http.Get(item.DownloadURL)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	filename := fmt.Sprintf("%s - %s.mp3",
		strings.TrimSpace(item.Artist),
		strings.TrimSpace(item.Title),
	)
	fd, err := os.Create(filename)
	if err != nil {
		return err
	}
	io.Copy(fd, response.Body)
	return nil
}

func main() {
	var (
		zingURL = flag.String("url", "", "Zing MP3 URL to be parsed")
	)
	flag.Parse()

	album, err := GetAlbum(*zingURL)
	log.Printf("%v %v", album, err)
	for _, item := range album.Items {
		log.Printf("%s - %s (%s)\n", item.Artist, item.Title, item.DownloadURL)
		DownloadAlbumItem(&item)
	}
}
