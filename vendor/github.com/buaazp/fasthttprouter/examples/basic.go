package main

import (
	"fmt"
	"log"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
)

// Index is the index handler
func Index(ctx *fasthttp.RequestCtx, _ fasthttprouter.Params) {
	fmt.Fprint(ctx, "Welcome!\n")
}

// Hello is the Hello handler
func Hello(ctx *fasthttp.RequestCtx, ps fasthttprouter.Params) {
	fmt.Fprintf(ctx, "hello, %s!\n", ps.ByName("name"))
}

func main() {
	router := fasthttprouter.New()
	router.GET("/", Index)
	router.GET("/hello/:name", Hello)

	log.Fatal(fasthttp.ListenAndServe(":8080", router.Handler))
}
