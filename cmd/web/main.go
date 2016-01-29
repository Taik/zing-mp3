package main

import (
	"archive/zip"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sync"

	"github.com/Taik/zing-mp3/zing"
	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
	log "gopkg.in/inconshreveable/log15.v2"
)

type albumJob struct {
	album         *zing.Album
	downloadQueue chan zing.AlbumItem
	downloadSync  *sync.WaitGroup
	zipQueue      chan zipFile
	zipSync       *sync.WaitGroup
	zipFile       *os.File
}

type zipFile struct {
	Filename string
	Buffer   io.Reader
}

func newAlbumJob(album *zing.Album) (*albumJob, error) {
	archive, err := ioutil.TempFile("/tmp", "album-")
	if err != nil {
		log.Error("Unable to create temp file", "error", err)
		return nil, err
	}

	return &albumJob{
		album:         album,
		downloadQueue: make(chan zing.AlbumItem),
		downloadSync:  &sync.WaitGroup{},
		zipQueue:      make(chan zipFile),
		zipSync:       &sync.WaitGroup{},
		zipFile:       archive,
	}, nil
}

func (a *albumJob) Start() {
	// Start N workers
	a.downloadSync.Add(4)
	for i := 0; i < 4; i++ {
		go a.startDownloader()
	}

	// Start Zipper
	a.zipSync.Add(1)
	go a.startZipper()

	for _, item := range a.album.Items {
		a.downloadQueue <- item
	}
	close(a.downloadQueue)
	a.downloadSync.Wait()

	close(a.zipQueue)
	a.zipSync.Wait()
}

func (a *albumJob) startDownloader() {
	defer a.downloadSync.Done()

	for item := range a.downloadQueue {
		log.Debug("Processing album item",
			"artist", item.Artist,
			"title", item.Title,
			"url", item.ItemURL,
		)

		buf, err := downloadURL(item.DownloadURL)
		if err != nil {
			log.Error("Unable to download item",
				"download_url", item.DownloadURL,
				"error", err,
			)
			return
		}
		a.zipQueue <- zipFile{
			Filename: item.Name(),
			Buffer:   buf,
		}
		log.Info("Processed album item",
			"download_url", item.DownloadURL,
			"filename", item.Name(),
		)
	}
}

func (a *albumJob) startZipper() {
	defer a.zipSync.Done()

	zipBuffer := zip.NewWriter(a.zipFile)
	defer zipBuffer.Close()

	for file := range a.zipQueue {
		filename := file.Filename
		log.Debug("Creating new item in archive",
			"filename", filename,
		)

		f, err := zipBuffer.Create(filename)
		if err != nil {
			log.Error("Unable to create new item in archive",
				"filename", filename,
			)
			return
		}

		log.Debug("Copying buffer into item",
			"filename", filename,
		)

		_, err = io.Copy(f, file.Buffer)
		if err != nil {
			log.Error("Unable to copy buffer into zip file",
				"filename", filename,
			)
			return
		}
		zipBuffer.Flush()
	}
	log.Debug("Archive completed")
}

func downloadURL(url string) (io.Reader, error) {
	buf := &bytes.Buffer{}
	log.Debug("Downloading item", "download_url", url)
	response, err := http.Get(url)
	if err != nil {
		log.Error("Unable to request album item", "download_url", url)
		return nil, err
	}
	defer response.Body.Close()
	io.Copy(buf, response.Body)

	return buf, nil
}

func zingAlbumHandler(ctx *fasthttp.RequestCtx, params fasthttprouter.Params) {
	zingURL := string(ctx.QueryArgs().Peek("url"))
	log.Info("Zing-mp3 album request",
		"zing_url", zingURL,
	)

	album, err := zing.ParseAlbumData(zingURL)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		fmt.Fprint(ctx, err)
		return
	}

	job, err := newAlbumJob(album)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		fmt.Fprint(ctx, err)
		return
	}

	job.Start()

	ctx.SetStatusCode(fasthttp.StatusOK)
	job.zipFile.Seek(0, 0)
	io.Copy(ctx, job.zipFile)
}

func main() {
	var (
		port = flag.Int("port", 8000, "Port to listen on")
	)
	flag.Parse()
	zing.Logger.SetHandler(log.LvlFilterHandler(log.LvlDebug, log.StdoutHandler))

	router := fasthttprouter.New()
	router.GET("/album/", zingAlbumHandler)
	fasthttp.ListenAndServe(fmt.Sprintf(":%d", *port), router.Handler)
}
