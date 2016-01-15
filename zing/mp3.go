package zing

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/Taik/zing-mp3/tags"
	"gopkg.in/inconshreveable/log15.v2"
)

var (
	// Logger is the logger instance to be used throughout the package
	Logger           = log15.New()
	errNoPlayerFound = errors.New("no HTML5 player instance found")
	errInvalidURL    = errors.New("invalid url")
)

func init() {
	// Initializes logger to throw away all messages by default
	Logger.SetHandler(log15.DiscardHandler())
}

// AlbumItem represents each item in Album.
type AlbumItem struct {
	Title       string `xml:"title"`
	Artist      string `xml:"performer"`
	ItemURL     string `xml:"link"`
	DownloadURL string `xml:"source"`
	LyricURL    string `xml:"lyric"`
}

// Album represents a Zing MP3 player source.
type Album struct {
	XMLName xml.Name    `xml:"data"`
	Items   []AlbumItem `xml:"item"`
}

// ParseAlbumData parses a zing MP3 URL and returns a Album associated with the current player on the page.
func ParseAlbumData(zingURL string) (*Album, error) {
	if zingURL == "" {
		Logger.Error("Invalid album data URL",
			"zing_url", zingURL,
		)
		return nil, errInvalidURL
	}
	Logger.Debug("Parsing for album data URL",
		"zing_url", zingURL,
	)

	doc, err := goquery.NewDocument(zingURL)
	if err != nil {
		return nil, err
	}

	dataXMLURL, found := doc.Find("div#html5player").Attr("data-xml")
	if found == false {
		return nil, errNoPlayerFound
	}

	Logger.Debug("Found zing album data URL",
		"album_data_xml", dataXMLURL,
	)
	response, err := http.Get(dataXMLURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	album := &Album{}
	err = xml.NewDecoder(response.Body).Decode(album)
	if err != nil {
		return nil, err
	}

	return album, nil
}

// DownloadAlbum initializes
func DownloadAlbum(zingURL string) error {
	album, err := ParseAlbumData(zingURL)
	if err != nil {
		return err
	}

	Logger.Debug("Found items to download",
		"item_count", len(album.Items),
		"album_url", zingURL,
	)
	for _, item := range album.Items {
		Logger.Info("Processing item",
			"artist", item.Artist,
			"title", item.Title,
			"download_url", item.DownloadURL,
		)

		Logger.Debug("Downloading item", "download_url", item.DownloadURL)
		fd, err := DownloadAlbumItem(&item)
		if err != nil {
			Logger.Error("Could not download item", "error", err)
		} else {
			Logger.Debug("File downloaded", "file_path", fd.Name())
		}

		Logger.Debug("Updating mp3 tags", "file_path", fd.Name())
		err = tags.UpdateMP3Tags(fd, item.Artist, item.Title)
		if err != nil {
			Logger.Error("Could not update mp3 tags", "file_path", fd.Name())
		} else {
			Logger.Debug("File mp3 tag updated", "file_path", fd.Name())
		}
	}

	return nil
}

// DownloadAlbumItem fetches the song from DownloadURL and returns an os.File which represents the file on-disk.
func DownloadAlbumItem(item *AlbumItem) (*os.File, error) {
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
