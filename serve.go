package goweb

import (
	"fmt"
	"github.com/protamail/htm"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
)

type Handler interface {
	HandleRequest(w http.ResponseWriter, req *http.Request) htm.Result
}

type RootMux struct {
	Handler Handler
}

func (rm *RootMux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("Error: %v\n%s", err, debug.Stack())
			fmt.Fprintf(w, "Error: %v\n%s", err, debug.Stack())
		}
	}()
	if rm.Handler != nil {
		w.Header().Set("Cache-Control", "no-store") //no caching unless handler overrides
		result := rm.Handler.HandleRequest(w, req)
		if !result.IsEmpty() {
			fmt.Fprint(w, result.String())
		}
	}
}

func CutPrefix(origPath string, pfxCount int) (ctxPrefix, routePath string) {
	routePath = origPath
	if pfxCount > 0 {
		if len(routePath) == 0 || routePath[0] != '/' {
			log.Panic("Invalid request path")
		}
		idx := 0
		for i := 0; i < pfxCount; i++ {
			si := strings.IndexByte(routePath[1:], '/')
			if si == -1 {
				log.Panicf("pfxCount=%d setting is too large for this request path: %s", pfxCount, origPath)
			}
			idx += si + 1
			ctxPrefix = origPath[:idx]
			routePath = origPath[idx:]
		}
	}
	return
}

func Redirect(w http.ResponseWriter, req *http.Request, location string) {
	if len(w.Header().Get("X-Ajax")) > 0 || w.Header().Get("X-Requested-With") == "XMLHttpRequest" {
		//don't redirect AJAX requests, let the caller handle new location at page level
		w.WriteHeader(http.StatusUnauthorized) //401 Unauthorized as the most likely cause
		w.Header().Set("X-Location", location)
	} else {
		http.Redirect(w, req, location, http.StatusFound) //302 Found
	}
}
