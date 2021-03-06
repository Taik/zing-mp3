package main

import (
	"flag"

	"github.com/Taik/zing-mp3/zing"
	log "gopkg.in/inconshreveable/log15.v2"
)

func main() {
	var (
		zingURL     = flag.String("url", "", "Zing MP3 URL to be parsed")
		downloadDir = flag.String("dir", ".", "Directory to download into")
	)
	flag.Parse()

	zing.Logger.SetHandler(log.LvlFilterHandler(log.LvlDebug, log.StdoutHandler))

	zing.DownloadAlbum(*zingURL, *downloadDir)
}
