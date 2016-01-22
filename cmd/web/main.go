package main

import (
	"archive/zip"
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/Taik/zing-mp3/zing"
	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
	log "gopkg.in/inconshreveable/log15.v2"
)

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

	ctx.SetStatusCode(fasthttp.StatusOK)

	zipBuffer := zip.NewWriter(ctx)
	zipLock := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	wg.Add(len(album.Items))

	for _, item := range album.Items {
		go func(item zing.AlbumItem) {
			defer wg.Done()
			buf := &bytes.Buffer{}

			log.Debug("Downloading item", "download_url", item.DownloadURL)
			response, err := http.Get(item.DownloadURL)
			if err != nil {
				log.Error("Unable to request album item", "download_url", item.DownloadURL)
				return
			}
			defer response.Body.Close()

			_, err = io.Copy(buf, response.Body)
			if err != nil {
				log.Error("Unable to copy buffer",
					"content_len", response.ContentLength,
				)
				return
			}

			filename := item.Name()
			log.Debug("Creating new item in archive",
				"filename", filename,
			)

			zipLock.Lock()
			defer zipLock.Unlock()

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

			_, err = io.Copy(f, buf)
			if err != nil {
				log.Error("Unable to copy buffer into zip file",
					"filename", filename,
				)
				return
			}

			log.Info("Processed album item",
				"download_url", item.DownloadURL,
				"filename", filename,
			)
			zipBuffer.Flush()
		}(item)
	}
	wg.Wait()
	zipBuffer.Close()
}

func main() {
	var (
		port = flag.Int("port", 8000, "Port to listen on")
	)
	zing.Logger.SetHandler(log.LvlFilterHandler(log.LvlDebug, log.StdoutHandler))

	router := fasthttprouter.New()
	router.GET("/zing-mp3/album/", zingAlbumHandler)
	fasthttp.ListenAndServe(fmt.Sprintf(":%d", *port), router.Handler)
}
