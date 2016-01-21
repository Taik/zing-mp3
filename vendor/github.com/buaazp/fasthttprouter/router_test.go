// Copyright 2013 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package fasthttprouter

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
)

func TestParams(t *testing.T) {
	ps := Params{
		Param{"param1", "value1"},
		Param{"param2", "value2"},
		Param{"param3", "value3"},
	}
	for i := range ps {
		if val := ps.ByName(ps[i].Key); val != ps[i].Value {
			t.Errorf("Wrong value for %s: Got %s; Want %s", ps[i].Key, val, ps[i].Value)
		}
	}
	if val := ps.ByName("noKey"); val != "" {
		t.Errorf("Expected empty string for not found key; got: %s", val)
	}
}

func TestRouter(t *testing.T) {
	router := New()

	routed := false
	router.Handle("GET", "/user/:name", func(ctx *fasthttp.RequestCtx, ps Params) {
		routed = true
		want := Params{Param{"name", "gopher"}}
		if !reflect.DeepEqual(ps, want) {
			t.Fatalf("wrong wildcard values: want %v, got %v", want, ps)
		}
		ctx.Success("foo/bar", []byte("success"))
	})

	s := &fasthttp.Server{
		Handler: router.Handler,
	}

	rw := &readWriter{}
	rw.r.WriteString("GET /user/gopher?baz HTTP/1.1\r\nHost: google.com\r\n\r\n")

	ch := make(chan error)
	go func() {
		ch <- s.ServeConn(rw)
	}()

	select {
	case err := <-ch:
		if err != nil {
			t.Fatalf("return error %s", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout")
	}

	if !routed {
		t.Fatal("routing failed")
	}
}

type handlerStruct struct {
	handeled *bool
}

func (h handlerStruct) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	*h.handeled = true
}

func TestRouterAPI(t *testing.T) {
	var get, head, options, post, put, patch, deleted bool

	router := New()
	router.GET("/GET", func(ctx *fasthttp.RequestCtx, _ Params) {
		get = true
	})
	router.HEAD("/GET", func(ctx *fasthttp.RequestCtx, _ Params) {
		head = true
	})
	router.OPTIONS("/GET", func(ctx *fasthttp.RequestCtx, _ Params) {
		options = true
	})
	router.POST("/POST", func(ctx *fasthttp.RequestCtx, _ Params) {
		post = true
	})
	router.PUT("/PUT", func(ctx *fasthttp.RequestCtx, _ Params) {
		put = true
	})
	router.PATCH("/PATCH", func(ctx *fasthttp.RequestCtx, _ Params) {
		patch = true
	})
	router.DELETE("/DELETE", func(ctx *fasthttp.RequestCtx, _ Params) {
		deleted = true
	})

	s := &fasthttp.Server{
		Handler: router.Handler,
	}

	rw := &readWriter{}
	ch := make(chan error)

	rw.r.WriteString("GET /GET HTTP/1.1\r\nHost: google.com\r\n\r\n")
	go func() {
		ch <- s.ServeConn(rw)
	}()
	select {
	case err := <-ch:
		if err != nil {
			t.Fatalf("return error %s", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout")
	}
	if !get {
		t.Error("routing GET failed")
	}

	rw.r.WriteString("HEAD /GET HTTP/1.1\r\nHost: google.com\r\n\r\n")
	go func() {
		ch <- s.ServeConn(rw)
	}()
	select {
	case err := <-ch:
		if err != nil {
			t.Fatalf("return error %s", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout")
	}
	if !head {
		t.Error("routing HEAD failed")
	}

	rw.r.WriteString("OPTIONS /GET HTTP/1.1\r\nHost: google.com\r\n\r\n")
	go func() {
		ch <- s.ServeConn(rw)
	}()
	select {
	case err := <-ch:
		if err != nil {
			t.Fatalf("return error %s", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout")
	}
	if !options {
		t.Error("routing OPTIONS failed")
	}

	rw.r.WriteString("POST /POST HTTP/1.1\r\nHost: google.com\r\n\r\n")
	go func() {
		ch <- s.ServeConn(rw)
	}()
	select {
	case err := <-ch:
		if err != nil {
			t.Fatalf("return error %s", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout")
	}
	if !post {
		t.Error("routing POST failed")
	}

	rw.r.WriteString("PUT /PUT HTTP/1.1\r\nHost: google.com\r\n\r\n")
	go func() {
		ch <- s.ServeConn(rw)
	}()
	select {
	case err := <-ch:
		if err != nil {
			t.Fatalf("return error %s", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout")
	}
	if !put {
		t.Error("routing PUT failed")
	}

	rw.r.WriteString("PATCH /PATCH HTTP/1.1\r\nHost: google.com\r\n\r\n")
	go func() {
		ch <- s.ServeConn(rw)
	}()
	select {
	case err := <-ch:
		if err != nil {
			t.Fatalf("return error %s", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout")
	}
	if !patch {
		t.Error("routing PATCH failed")
	}

	rw.r.WriteString("DELETE /DELETE HTTP/1.1\r\nHost: google.com\r\n\r\n")
	go func() {
		ch <- s.ServeConn(rw)
	}()
	select {
	case err := <-ch:
		if err != nil {
			t.Fatalf("return error %s", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout")
	}
	if !deleted {
		t.Error("routing DELETE failed")
	}
}

func TestRouterRoot(t *testing.T) {
	router := New()
	recv := catchPanic(func() {
		router.GET("noSlashRoot", nil)
	})
	if recv == nil {
		t.Fatal("registering path not beginning with '/' did not panic")
	}
}

func TestRouterChaining(t *testing.T) {
	router1 := New()
	router2 := New()
	router1.NotFound = router2.Handler

	fooHit := false
	router1.POST("/foo", func(ctx *fasthttp.RequestCtx, _ Params) {
		fooHit = true
		ctx.SetStatusCode(fasthttp.StatusOK)
	})

	barHit := false
	router2.POST("/bar", func(ctx *fasthttp.RequestCtx, _ Params) {
		barHit = true
		ctx.SetStatusCode(fasthttp.StatusOK)
	})

	s := &fasthttp.Server{
		Handler: router1.Handler,
	}

	rw := &readWriter{}
	ch := make(chan error)

	rw.r.WriteString("POST /foo HTTP/1.1\r\nHost: google.com\r\n\r\n")
	go func() {
		ch <- s.ServeConn(rw)
	}()
	select {
	case err := <-ch:
		if err != nil {
			t.Fatalf("return error %s", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout")
	}
	br := bufio.NewReader(&rw.w)
	var resp fasthttp.Response
	if err := resp.Read(br); err != nil {
		t.Fatalf("Unexpected error when reading response: %s", err)
	}
	if !(resp.Header.StatusCode() == fasthttp.StatusOK && fooHit) {
		t.Errorf("Regular routing failed with router chaining.")
		t.FailNow()
	}

	rw.r.WriteString("POST /bar HTTP/1.1\r\nHost: google.com\r\n\r\n")
	go func() {
		ch <- s.ServeConn(rw)
	}()
	select {
	case err := <-ch:
		if err != nil {
			t.Fatalf("return error %s", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout")
	}
	if err := resp.Read(br); err != nil {
		t.Fatalf("Unexpected error when reading response: %s", err)
	}
	if !(resp.Header.StatusCode() == fasthttp.StatusOK && barHit) {
		t.Errorf("Chained routing failed with router chaining.")
		t.FailNow()
	}

	rw.r.WriteString("POST /qax HTTP/1.1\r\nHost: google.com\r\n\r\n")
	go func() {
		ch <- s.ServeConn(rw)
	}()
	select {
	case err := <-ch:
		if err != nil {
			t.Fatalf("return error %s", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout")
	}
	if err := resp.Read(br); err != nil {
		t.Fatalf("Unexpected error when reading response: %s", err)
	}
	if !(resp.Header.StatusCode() == fasthttp.StatusNotFound) {
		t.Errorf("NotFound behavior failed with router chaining.")
		t.FailNow()
	}
}

func TestRouterNotAllowed(t *testing.T) {
	handlerFunc := func(_ *fasthttp.RequestCtx, _ Params) {}

	router := New()
	router.POST("/path", handlerFunc)

	// Test not allowed
	s := &fasthttp.Server{
		Handler: router.Handler,
	}

	rw := &readWriter{}
	ch := make(chan error)

	rw.r.WriteString("GET /path HTTP/1.1\r\nHost: google.com\r\n\r\n")
	go func() {
		ch <- s.ServeConn(rw)
	}()
	select {
	case err := <-ch:
		if err != nil {
			t.Fatalf("return error %s", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout")
	}
	br := bufio.NewReader(&rw.w)
	var resp fasthttp.Response
	if err := resp.Read(br); err != nil {
		t.Fatalf("Unexpected error when reading response: %s", err)
	}
	if !(resp.Header.StatusCode() == fasthttp.StatusMethodNotAllowed) {
		t.Errorf("NotAllowed handling failed: Code=%d, Header=%v", resp.Header.StatusCode(), resp.Header)
	}

	responseText := "custom method"
	router.MethodNotAllowed = fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusTeapot)
		ctx.Write([]byte(responseText))
	})
	rw.r.WriteString("GET /path HTTP/1.1\r\nHost: google.com\r\n\r\n")
	go func() {
		ch <- s.ServeConn(rw)
	}()
	select {
	case err := <-ch:
		if err != nil {
			t.Fatalf("return error %s", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout")
	}
	if err := resp.Read(br); err != nil {
		t.Fatalf("Unexpected error when reading response: %s", err)
	}
	if !bytes.Equal(resp.Body(), []byte(responseText)) {
		t.Errorf("unexpected response got %q want %q", string(resp.Body()), responseText)
	}
	if resp.Header.StatusCode() != fasthttp.StatusTeapot {
		t.Errorf("unexpected response code %d want %d", resp.Header.StatusCode(), fasthttp.StatusTeapot)
	}
}

func TestRouterNotFound(t *testing.T) {
	handlerFunc := func(_ *fasthttp.RequestCtx, _ Params) {}

	router := New()
	router.GET("/path", handlerFunc)
	router.GET("/dir/", handlerFunc)
	router.GET("/", handlerFunc)

	testRoutes := []struct {
		route string
		code  int
	}{
		{"/path/", 301},   // TSR -/
		{"/dir", 301},     // TSR +/
		{"/", 200},        // TSR +/
		{"/PATH", 301},    // Fixed Case
		{"/DIR", 301},     // Fixed Case
		{"/PATH/", 301},   // Fixed Case -/
		{"/DIR/", 301},    // Fixed Case +/
		{"/../path", 200}, // CleanPath
		{"/nope", 404},    // NotFound
	}

	s := &fasthttp.Server{
		Handler: router.Handler,
	}

	rw := &readWriter{}
	br := bufio.NewReader(&rw.w)
	var resp fasthttp.Response
	ch := make(chan error)
	for _, tr := range testRoutes {
		rw.r.WriteString(fmt.Sprintf("GET %s HTTP/1.1\r\nHost: google.com\r\n\r\n", tr.route))
		go func() {
			ch <- s.ServeConn(rw)
		}()
		select {
		case err := <-ch:
			if err != nil {
				t.Fatalf("return error %s", err)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("timeout")
		}
		if err := resp.Read(br); err != nil {
			t.Fatalf("Unexpected error when reading response: %s", err)
		}
		if !(resp.Header.StatusCode() == tr.code) {
			t.Errorf("NotFound handling route %s failed: Code=%d want=%d",
				tr.route, resp.Header.StatusCode(), tr.code)
		}
	}

	// Test custom not found handler
	var notFound bool
	router.NotFound = fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(404)
		notFound = true
	})
	rw.r.WriteString("GET /nope HTTP/1.1\r\nHost: google.com\r\n\r\n")
	go func() {
		ch <- s.ServeConn(rw)
	}()
	select {
	case err := <-ch:
		if err != nil {
			t.Fatalf("return error %s", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout")
	}
	if err := resp.Read(br); err != nil {
		t.Fatalf("Unexpected error when reading response: %s", err)
	}
	if !(resp.Header.StatusCode() == 404 && notFound == true) {
		t.Errorf("Custom NotFound handler failed: Code=%d, Header=%v", resp.Header.StatusCode(), string(resp.Header.Peek("Location")))
	}

	// Test other method than GET (want 307 instead of 301)
	router.PATCH("/path", handlerFunc)
	rw.r.WriteString("PATCH /path/ HTTP/1.1\r\nHost: google.com\r\n\r\n")
	go func() {
		ch <- s.ServeConn(rw)
	}()
	select {
	case err := <-ch:
		if err != nil {
			t.Fatalf("return error %s", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout")
	}
	if err := resp.Read(br); err != nil {
		t.Fatalf("Unexpected error when reading response: %s", err)
	}
	if !(resp.Header.StatusCode() == 307) {
		t.Errorf("Custom NotFound handler failed: Code=%d, Header=%v", resp.Header.StatusCode(), string(resp.Header.Peek("Location")))
	}

	// Test special case where no node for the prefix "/" exists
	router = New()
	router.GET("/a", handlerFunc)
	s.Handler = router.Handler
	rw.r.WriteString("GET / HTTP/1.1\r\nHost: google.com\r\n\r\n")
	go func() {
		ch <- s.ServeConn(rw)
	}()
	select {
	case err := <-ch:
		if err != nil {
			t.Fatalf("return error %s", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout")
	}
	if err := resp.Read(br); err != nil {
		t.Fatalf("Unexpected error when reading response: %s", err)
	}
	if !(resp.Header.StatusCode() == 404) {
		t.Errorf("NotFound handling route / failed: Code=%d", resp.Header.StatusCode())
	}
}

func TestRouterPanicHandler(t *testing.T) {
	router := New()
	panicHandled := false

	router.PanicHandler = func(ctx *fasthttp.RequestCtx, p interface{}) {
		panicHandled = true
	}

	router.Handle("PUT", "/user/:name", func(_ *fasthttp.RequestCtx, _ Params) {
		panic("oops!")
	})

	defer func() {
		if rcv := recover(); rcv != nil {
			t.Fatal("handling panic failed")
		}
	}()

	s := &fasthttp.Server{
		Handler: router.Handler,
	}

	rw := &readWriter{}
	ch := make(chan error)

	rw.r.WriteString(string("PUT /user/gopher HTTP/1.1\r\nHost: google.com\r\n\r\n"))
	go func() {
		ch <- s.ServeConn(rw)
	}()
	select {
	case err := <-ch:
		if err != nil {
			t.Fatalf("return error %s", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout")
	}

	if !panicHandled {
		t.Fatal("simulating failed")
	}
}

func TestRouterLookup(t *testing.T) {
	routed := false
	wantHandle := func(_ *fasthttp.RequestCtx, _ Params) {
		routed = true
	}
	wantParams := Params{Param{"name", "gopher"}}

	router := New()

	// try empty router first
	handle, _, tsr := router.Lookup("GET", "/nope")
	if handle != nil {
		t.Fatalf("Got handle for unregistered pattern: %v", handle)
	}
	if tsr {
		t.Error("Got wrong TSR recommendation!")
	}

	// insert route and try again
	router.GET("/user/:name", wantHandle)

	handle, params, tsr := router.Lookup("GET", "/user/gopher")
	if handle == nil {
		t.Fatal("Got no handle!")
	} else {
		handle(nil, nil)
		if !routed {
			t.Fatal("Routing failed!")
		}
	}

	if !reflect.DeepEqual(params, wantParams) {
		t.Fatalf("Wrong parameter values: want %v, got %v", wantParams, params)
	}

	handle, _, tsr = router.Lookup("GET", "/user/gopher/")
	if handle != nil {
		t.Fatalf("Got handle for unregistered pattern: %v", handle)
	}
	if !tsr {
		t.Error("Got no TSR recommendation!")
	}

	handle, _, tsr = router.Lookup("GET", "/nope")
	if handle != nil {
		t.Fatalf("Got handle for unregistered pattern: %v", handle)
	}
	if tsr {
		t.Error("Got wrong TSR recommendation!")
	}
}

type mockFileSystem struct {
	opened bool
}

func (mfs *mockFileSystem) Open(name string) (http.File, error) {
	mfs.opened = true
	return nil, errors.New("this is just a mock")
}

func TestRouterServeFiles(t *testing.T) {
	router := New()

	recv := catchPanic(func() {
		router.ServeFiles("/noFilepath", os.TempDir())
	})
	if recv == nil {
		t.Fatal("registering path not ending with '*filepath' did not panic")
	}
	body := []byte("fake ico")
	ioutil.WriteFile(os.TempDir()+"/favicon.ico", body, 0644)

	router.ServeFiles("/*filepath", os.TempDir())

	s := &fasthttp.Server{
		Handler: router.Handler,
	}

	rw := &readWriter{}
	ch := make(chan error)

	rw.r.WriteString(string("GET /favicon.ico HTTP/1.1\r\nHost: google.com\r\n\r\n"))
	go func() {
		ch <- s.ServeConn(rw)
	}()
	select {
	case err := <-ch:
		if err != nil {
			t.Fatalf("return error %s", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout")
	}

	br := bufio.NewReader(&rw.w)
	var resp fasthttp.Response
	if err := resp.Read(br); err != nil {
		t.Fatalf("Unexpected error when reading response: %s", err)
	}
	if resp.Header.StatusCode() != 200 {
		t.Fatalf("Unexpected status code %d. Expected %d", resp.Header.StatusCode(), 423)
	}
	if !bytes.Equal(resp.Body(), body) {
		t.Fatalf("Unexpected body %q. Expected %q", resp.Body(), string(body))
	}
}

type readWriter struct {
	net.Conn
	r bytes.Buffer
	w bytes.Buffer
}

var zeroTCPAddr = &net.TCPAddr{
	IP: net.IPv4zero,
}

func (rw *readWriter) Close() error {
	return nil
}

func (rw *readWriter) Read(b []byte) (int, error) {
	return rw.r.Read(b)
}

func (rw *readWriter) Write(b []byte) (int, error) {
	return rw.w.Write(b)
}

func (rw *readWriter) RemoteAddr() net.Addr {
	return zeroTCPAddr
}

func (rw *readWriter) LocalAddr() net.Addr {
	return zeroTCPAddr
}

func (rw *readWriter) SetReadDeadline(t time.Time) error {
	return nil
}

func (rw *readWriter) SetWriteDeadline(t time.Time) error {
	return nil
}
