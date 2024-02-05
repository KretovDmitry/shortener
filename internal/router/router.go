package router

import (
	"net/http"
	"regexp"
	"slices"
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
