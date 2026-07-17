package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/idena-network/idena-indexer/log"
)

const (
	maxRequestBodyBytes = 1 << 20
	maxHeaderBytes      = 1 << 20
	readHeaderTimeout   = 10 * time.Second
	readTimeout         = 30 * time.Second
	writeTimeout        = 2 * time.Minute
	idleTimeout         = 2 * time.Minute
)

func NewServer(
	port int,
	logger log.Logger,
) *Server {
	return &Server{
		port: port,
		log:  logger,
	}
}

type Server struct {
	port    int
	counter int
	log     log.Logger
	mutex   sync.Mutex
}

func (s *Server) Start(routerInitializers ...RouterInitializer) {
	router := mux.NewRouter()
	apiRouter := router.PathPrefix("/api").Subrouter()

	for _, ri := range routerInitializers {
		ri.InitRouter(apiRouter)
	}

	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})
	handler := handlers.CORS(originsOk, headersOk, methodsOk)(s.requestFilter(apiRouter))
	err := newHTTPServer(s.port, handler).ListenAndServe()
	if err != nil {
		panic(err)
	}
}

func newHTTPServer(port int, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}
}

func (s *Server) generateReqId() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	id := s.counter
	s.counter++
	return id
}

func (s *Server) requestFilter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqId := s.generateReqId()
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		}
		s.log.Debug("Got api request", "reqId", reqId)
		err := r.ParseForm()
		if err != nil {
			s.log.Error("Unable to parse API request", "reqId", reqId)
			var maxBytesError *http.MaxBytesError
			if errors.As(err, &maxBytesError) {
				http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer s.log.Debug("Completed api request", "reqId", reqId)
		for name, value := range r.Form {
			r.Form[strings.ToLower(name)] = value
		}
		r.URL.Path = strings.ToLower(r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
