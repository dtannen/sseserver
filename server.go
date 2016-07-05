package sseserver

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/GeertJohan/go.rice"
	. "github.com/azer/debug"
	"github.com/garyburd/redigo/redis"
)

// Interface to a SSE server.
//
// Exposes a send-only chan `broadcast`, any SSEMessage sent to this channel
// will be broadcast out to any connected clients subscribed to a namespace
// that matches the message.
type Server struct {
	Broadcast chan<- SSEMessage
	hub       *hub
}

// Creates a new Server and returns a reference to it.
func NewServer() *Server {

	// set up the public interface
	var s = Server{}

	// start up our actual internal connection hub
	// which we keep in the server struct as private
	var h = newHub()
	s.hub = h
	go h.run()

	// expose just the broadcast chanel to public
	// will be typecast to send-only
	s.Broadcast = h.broadcast

	// return handle
	return &s
}

// Begin serving connections on specified address.
//
// This method blocks forever, as it is basically a setup wrapper around
// http.ListenAndServe()
func (s *Server) Serve(addr string) {

	// get redis pool
	redisHost := os.Getenv("REDIS_HOST")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	if redisHost == "" {
		redisHost = "localhost:6379"
	}
	if redisPassword == "" {
		redisPassword = ""
	}
	pool := newPool(redisHost, redisPassword)
	// set up routes.
	// use an anonymous function for closure in order to pass value to handler
	// https://groups.google.com/forum/#!topic/golang-nuts/SGn1gd290zI
	http.HandleFunc("/subscribe/", func(w http.ResponseWriter, r *http.Request) {
		sseHandler(w, r, s.hub, pool)
	})

	http.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		// kinda ridiculous workaround for serving a single static file, sigh.
		// works for now without changing paths tho...
		box, err := rice.FindBox("views")
		if err != nil {
			log.Fatalf("error opening rice.Box: %s\n", err)
		}

		file, err := box.Open("admin.html")
		if err != nil {
			log.Fatalf("could not open file: %s\n", err)
		}

		fstat, err := file.Stat()
		if err != nil {
			log.Fatalf("could not stat file: %s\n", err)
		}

		http.ServeContent(w, r, fstat.Name(), fstat.ModTime(), file)
	})

	http.HandleFunc("/admin/status.json", func(w http.ResponseWriter, r *http.Request) {
		adminStatusDataHandler(w, r, s.hub)
	})

	// actually start the HTTP server
	Debug("Starting server on addr " + addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func newPool(host, password string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     2,
		MaxActive:   5,
		IdleTimeout: 5 * time.Second,
		Wait:        true,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", host)
			if err != nil {
				log.Println(err)
				return nil, err
			}
			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
					log.Println(err)
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
	}
}
