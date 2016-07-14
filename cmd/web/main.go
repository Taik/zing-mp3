package main

import (
	"archive/zip"
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"sync"

	"github.com/Taik/zing-mp3/zing"
	"github.com/buaazp/fasthttprouter"
	"github.com/oxtoacart/bpool"
	"github.com/valyala/fasthttp"
	log "gopkg.in/inconshreveable/log15.v2"
)

type albumJob struct {
	album         *zing.Album
	downloadQueue chan zing.AlbumItem
	downloadSync  *sync.WaitGroup
	zipQueue      chan zipFile
	zipSync       *sync.WaitGroup
	zipWriter     io.Writer

	bufferPool *bpool.BufferPool
}

type zipFile struct {
	Filename string
	Buffer   *bytes.Buffer
}

func newAlbumJob(album *zing.Album, out io.Writer) (*albumJob, error) {
	return &albumJob{
		album:         album,
		downloadQueue: make(chan zing.AlbumItem),
		downloadSync:  &sync.WaitGroup{},
		zipQueue:      make(chan zipFile, 2),
		zipSync:       &sync.WaitGroup{},
		zipWriter:     out,
		bufferPool:    bpool.NewBufferPool(12),
	}, nil
}

func (a *albumJob) Run() {
	// Start N workers
	a.downloadSync.Add(8)
	for i := 0; i < 8; i++ {
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
		buf := a.bufferPool.Get()

		log.Debug("Processing album item",
			"artist", item.Artist,
			"title", item.Title,
			"url", item.ItemURL,
		)

		err := downloadURL(buf, item.DownloadURL)
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

	zipBuffer := zip.NewWriter(a.zipWriter)
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

		log.Debug("Copying buffer into zip file",
			"filename", filename,
		)

		_, err = io.Copy(f, file.Buffer)
		if err != nil {
			log.Error("Unable to copy buffer into zip file",
				"filename", filename,
			)
			return
		}
		a.bufferPool.Put(file.Buffer)
		zipBuffer.Flush()
	}
	log.Debug("Archive completed")
}

func downloadURL(buf *bytes.Buffer, url string) error {
	log.Debug("Downloading item", "download_url", url)
	response, err := http.Get(url)
	if err != nil {
		log.Error("Unable to request album item", "download_url", url)
		return err
	}
	defer response.Body.Close()
	io.Copy(buf, response.Body)

	return nil
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

	job, err := newAlbumJob(album, ctx.Response.BodyWriter())
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		fmt.Fprint(ctx, err)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	job.Run()
}

func main() {
	var (
		port = flag.Int("port", 8000, "Port to listen on")
	)
	flag.Parse()
	zing.Logger.SetHandler(log.LvlFilterHandler(log.LvlDebug, log.StdoutHandler))

	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()

	router := fasthttprouter.New()
	router.GET("/album/", zingAlbumHandler)
	fasthttp.ListenAndServe(fmt.Sprintf(":%d", *port), router.Handler)
}
