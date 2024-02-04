// Package handler provides handlers.
package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"

	"github.com/KretovDmitry/shortener/internal/db"
)

type RouteEntry struct {
	Path        *regexp.Regexp
	Method      string
	ContentType string
	Handler     http.HandlerFunc
}

type Router struct {
	routes []RouteEntry
}

func (rtr *Router) Route(path *regexp.Regexp, method, contentType string, handlerFunc http.HandlerFunc) {
	e := RouteEntry{
		Path:        path,
		Method:      method,
		ContentType: contentType,
		Handler:     handlerFunc,
	}
	rtr.routes = append(rtr.routes, e)
}

func (re *RouteEntry) Match(r *http.Request) bool {
	if r.Method != re.Method {
		return false
	}

	if r.Header.Get("content-type") != re.ContentType {
		return false
	}

	return re.Path.MatchString(r.URL.Path)
}

func (rtr *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, e := range rtr.routes {
		match := e.Match(r)
		if !match {
			continue
		}

		e.Handler.ServeHTTP(w, r)
		return
	}

	// No matches, so it's a 400
	http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
}

var HomeRegexp = regexp.MustCompile(`^\/$`)

func CreateShortUrl(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("could't read body: %s\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	shortLink, err := db.SaveUrlMapping(string(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := "http://" + r.Host + "/" + shortLink

	w.Header().Set("content-type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(resp))
}

var Base58Regexp = regexp.MustCompile(`^\/[A-HJ-NP-Za-km-z1-9]{8}$`)

func HandleShortUrlRedirecth(w http.ResponseWriter, r *http.Request) {
	url, found := db.RetrieveInitialUrl(r.URL.Path[1:])
	if !found {
		msg := fmt.Sprintf("no such short URL: %s\n", r.URL.Path[1:])
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	w.Header().Set("location", url)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
