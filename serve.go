package goweb

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
	"io"

	"github.com/protamail/goweb/conf"
	"github.com/protamail/goweb/htm"
)

func Debug(d bool) {
	conf.Debug = d
}

type Handler interface {
	HandleRequest(w http.ResponseWriter, req *http.Request) htm.Result
}

type RootMux struct {
	Handler Handler
}

type ClientError struct {
	Msg string
}
func (v ClientError) Error() string {
	return v.Msg
}

func (rm *RootMux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var startTime time.Time
	if conf.Debug {
		startTime = time.Now()
	}
	defer func() {
		//for fine grain control, override this in the app handler
		if err := recover(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			stack := fmt.Sprintf("Error: %v\n%s", err, debug.Stack())
			_, ok := err.(ClientError)
			if ok {
				fmt.Fprintf(w, "%v", err)
			} else {
				if !conf.Debug {
					fmt.Fprint(w, "Server Error")
				} else {
					fmt.Fprint(w, stack)
				}
				log.Print(stack)
			}
		}
	}()
	if rm.Handler != nil {
		w.Header().Set("Cache-Control", "no-store") //no caching unless handler overrides
		req.ParseForm() //make req.Form values available
		result := rm.Handler.HandleRequest(w, req)
		if !result.IsEmpty() {
			fmt.Fprint(w, result.String())
		}
	}
	if conf.Debug {
		log.Printf("Finished %s %s", req.URL.RequestURI(), time.Now().Sub(startTime))
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

func IsAJAX(req *http.Request) bool {
	return len(req.Header.Get("X-Ajax")) > 0 || req.Header.Get("X-Requested-With") == "XMLHttpRequest"
}

func Redirect(w http.ResponseWriter, req *http.Request, location string) {
	if IsAJAX(req) {
		//don't redirect AJAX requests, let the caller handle new location at page level
		w.WriteHeader(http.StatusUnauthorized) //401 Unauthorized as the most likely cause
		w.Header().Set("X-Location", location)
	} else {
		http.Redirect(w, req, location, http.StatusFound) //302 Found
	}
}

func ReadBodyBytes(w http.ResponseWriter, req *http.Request, maxSize int64) (result []byte) {
	maxReader := http.MaxBytesReader(w, req.Body, maxSize)
	result, err := io.ReadAll(maxReader)
	if err != nil {
		log.Panic(err)
	}
	return
}
