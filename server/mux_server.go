package server

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	log "github.com/idleuncle/yados/utils/logger"
)

// MuxConn -
// **************** MuxConn ****************
type MuxConn struct {
	net.Conn
}

// NewMuxConn ()
// ======== NewMuxConn() ========
func NewMuxConn(conn net.Conn) *MuxConn {
	return &MuxConn{Conn: conn}
}

// MuxListener -
// **************** MuxListener ****************
type MuxListener struct {
	net.Listener
	config *tls.Config
}

// Accept ()
// ======== MuxListener::Accept() ========
func (l *MuxListener) Accept() (net.Conn, error) {

	conn, err := l.Listener.Accept()
	if err != nil {
		return conn, err
	}
	muxConn := NewMuxConn(conn)

	return muxConn, nil
}

// Close ()
// ======== MuxListener::Close() ========
func (l *MuxListener) Close() error {
	if l == nil {
		return nil
	}
	return l.Listener.Close()
}

// MuxServer -
// **************** MuServer ****************
type MuxServer struct {
	Name string
	http.Server
	listener        *MuxListener
	WaitGroup       *sync.WaitGroup
	GracefulTimeout time.Duration
	mutex           sync.Mutex
	closed          bool
	conns           map[net.Conn]http.ConnState
}

// NewMuxServer ()
// ======== NewMuxServer() ========
func NewMuxServer(name string, addr string, handler http.Handler) *MuxServer {
	ms := &MuxServer{
		Name: name,
		Server: http.Server{
			Addr:           addr,
			Handler:        handler,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		WaitGroup:       &sync.WaitGroup{},
		GracefulTimeout: 5 * time.Second,
	}

	ms.connState()

	return ms
}

// ListenAndServe ()
// ======== MuxServer::ListenAndServe() ========
func (ms *MuxServer) ListenAndServe() error {
	listener, err := net.Listen("tcp", ms.Server.Addr)
	if err != nil {
		return err
	}

	muxListener := &MuxListener{Listener: listener, config: &tls.Config{}}

	ms.mutex.Lock()
	ms.listener = muxListener
	ms.mutex.Unlock()

	log.Infof("Server "+ms.Name+" ListenAndServer(). addr:%s", ms.Server.Addr)

	return ms.Server.Serve(muxListener)
}

// ListenAndServeTLS ()
// ======== MuxServer::ListenAndServeTLS() ========
func (ms *MuxServer) ListenAndServeTLS(certFile, keyFile string) error {
	listener, err := net.Listen("tcp", ms.Server.Addr)
	if err != nil {
		return err
	}

	config := &tls.Config{}
	if config.NextProtos == nil {
		config.NextProtos = []string{"http/1.1", "h2"}
	}
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}

	muxListener := &MuxListener{Listener: listener, config: &tls.Config{}}

	ms.mutex.Lock()
	ms.listener = muxListener
	ms.mutex.Unlock()

	err = http.Serve(muxListener,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.TLS == nil {
				u := url.URL{
					Scheme:   "https",
					Opaque:   r.URL.Opaque,
					User:     r.URL.User,
					Host:     r.Host,
					Path:     r.URL.Path,
					RawQuery: r.URL.RawQuery,
					Fragment: r.URL.Fragment,
				}
				http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
			} else {
				ms.Server.Handler.ServeHTTP(w, r)
			}
		}))
	return err
}

// Close ()
// ======== MuxServer::Close() ========
func (ms *MuxServer) Close() error {
	ms.mutex.Lock()

	if ms.closed {
		return errors.New("Server has been closed.")
	}
	ms.closed = true

	if err := ms.listener.Close(); err != nil {
		return err
	}

	ms.SetKeepAlivesEnabled(false)
	for c, st := range ms.conns {
		// Force close any idle and new connections.
		// Waiting for other connections to close on their own (within the timout period)
		if st == http.StateIdle || st == http.StateNew {
			c.Close()
		}
	}

	// If the GracefulTimeout happens then forcefully close all connections.
	t := time.AfterFunc(ms.GracefulTimeout, func() {
		for c := range ms.conns {
			c.Close()
		}
	})
	defer t.Stop()

	ms.mutex.Unlock()

	// Block until all connetions are closed.
	ms.WaitGroup.Wait()
	return nil
}

// -------- MuxServer::connState() --------
func (ms *MuxServer) connState() {
	ms.Server.ConnState = func(c net.Conn, cs http.ConnState) {
		ms.mutex.Lock()
		defer ms.mutex.Unlock()

		switch cs {
		case http.StateNew:
			ms.WaitGroup.Add(1)
			if ms.conns == nil {
				ms.conns = make(map[net.Conn]http.ConnState)
			}
			ms.conns[c] = cs
		case http.StateActive:
			if _, ok := ms.conns[c]; ok {
				ms.conns[c] = cs
			}
		case http.StateIdle:
			if _, ok := ms.conns[c]; ok {
				ms.conns[c] = cs
			}
			if ms.closed {
				c.Close()
			}
		case http.StateHijacked, http.StateClosed:
			ms.forgetConn(c)
		}
	}
}

// -------- MuxServer::forgetConn() --------
func (ms *MuxServer) forgetConn(c net.Conn) {
	if _, ok := ms.conns[c]; ok {
		delete(ms.conns, c)
		ms.WaitGroup.Done()
	}
}
