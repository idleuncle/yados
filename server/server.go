package server

import (
	"net/http"

	"github.com/gorilla/mux"
)

// ======== Public Const Variables ========
const (
	SERVER_DEFAULT_NAME     = "yados"
	SERVER_DEFAULT_IP       = "0.0.0.0"
	SERVER_DEFAULT_PORT     = 8709
	SERVER_DEFAULT_STOREDIR = "yados.store"
)

// HandlerFunc - useful to chain different middleware http.Handler
type HandlerFunc func(http.Handler) http.Handler

func RegisterHandlers(r *mux.Router, handlerFns ...HandlerFunc) http.Handler {
	var f http.Handler
	f = r
	for _, hFn := range handlerFns {
		f = hFn(f)
	}
	return f
}
