// Package handler provides handlers.
package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"slices"

	"github.com/KretovDmitry/shortener/internal/db"
)

type Router struct {
	routes []RouteEntry
}

type RouteEntry struct {
	Path        *regexp.Regexp
	Method      string
	ContentType *[]string
	Handler     http.HandlerFunc
}

func (rtr *Router) Route(path *regexp.Regexp, method string, contentType *[]string, handlerFunc http.HandlerFunc) {
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

	if slices.Contains(*re.ContentType, r.Header.Get("content-type")) {
		return true
	}

	return re.Path.MatchString(r.URL.Path)
}

func (rtr *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, e := range rtr.routes {
		if !e.Match(r) {
			continue
		}
		e.Handler.ServeHTTP(w, r)
		return
	}

	// No matches, so it's a 400
	http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
}

var HomeRegexp = regexp.MustCompile(`^\/$`)

func CreateShortURL(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("could't read body: %s\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	shortLink, err := db.SaveURLMapping(string(body))
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

func HandleShortURLRedirect(w http.ResponseWriter, r *http.Request) {
	url, found := db.RetrieveInitialURL(r.URL.Path[1:])
	if !found {
		msg := fmt.Sprintf("no such short URL: %s\n", r.URL.Path[1:])
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	w.Header().Set("location", url)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
