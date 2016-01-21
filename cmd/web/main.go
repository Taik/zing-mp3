package main

import (
	"flag"
	"fmt"

	"github.com/Taik/zing-mp3/zing"
	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
	log "gopkg.in/inconshreveable/log15.v2"
)

func zingAlbumHandler(ctx *fasthttp.RequestCtx, params fasthttprouter.Params) {
	zingURL := params.ByName("albumURL")
	log.Info("Zing-mp3 album request",
		"zing_url", zingURL,
	)

	album, err := zing.ParseAlbumData(zingURL[1:len(zingURL)])
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		fmt.Fprint(ctx, err)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	fmt.Fprintf(ctx, "zing_url: %s, item_count: %d", zingURL, len(album.Items))
}

func main() {
	var (
		port = flag.Int("port", 8000, "Port to listen on")
	)
	zing.Logger.SetHandler(log.LvlFilterHandler(log.LvlDebug, log.StdoutHandler))

	router := fasthttprouter.New()
	router.GET("/zing-mp3/album/*albumURL", zingAlbumHandler)
	fasthttp.ListenAndServe(fmt.Sprintf(":%d", *port), router.Handler)
}
