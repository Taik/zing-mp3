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
	fmt.Fprintf(ctx, "zing_url: %s, item_count: %d\n", zingURL, len(album.Items))

	for i, item := range album.Items {
		fmt.Fprintf(ctx, "%d. %s - %s\n",
			i,
			item.Artist,
			item.Title,
		)
	}

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
