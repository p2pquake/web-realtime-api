package server

import (
	"context"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gopkg.in/olahol/melody.v1"
)

const maxConnectionsPerIP = 2

type HTTP struct {
	Done chan struct{}

	ctx           context.Context
	m             *melody.Melody
	ipConnections map[string]int
	ipMu          sync.Mutex
}

func clientIP(r *http.Request) string {
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-Ip"); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (s *HTTP) Start(ctx context.Context, bindTo string) {
	s.Done = make(chan struct{}, 1)
	s.ctx = ctx
	s.ipConnections = make(map[string]int)

	g := gin.Default()
	g.Use(cors.Default())

	m := melody.New()
	s.m = m

	m.HandleConnect(func(s *melody.Session) {
		ip := clientIP(s.Request)
		log.Printf("Connection established.\tAddr:%s\tUser-Agent:%s\n", ip, s.Request.Header["User-Agent"])
	})

	m.HandleDisconnect(func(s *melody.Session) {
		ip := clientIP(s.Request)
		log.Printf("Connection closed.\tAddr:%s\n", ip)
	})

	m.HandleError(func(s *melody.Session, e error) {
		ip := clientIP(s.Request)
		log.Printf("Error occured.\tAddr:%s\tError:%#v\n", ip, e)
	})

	m.HandleSentMessage(func(s *melody.Session, b []byte) {
		ip := clientIP(s.Request)
		log.Printf("Message sent.\tAddr:%s\n", ip)
	})

	g.GET("/", func(c *gin.Context) {
		c.String(200, "Hello world!")
	})

	v2 := g.Group("/v2")
	{
		v2.GET("/ws", func(c *gin.Context) {
			ip := clientIP(c.Request)

			s.ipMu.Lock()
			if s.ipConnections[ip] >= maxConnectionsPerIP {
				s.ipMu.Unlock()
				log.Printf("Connection rejected (too many connections).\tIP:%s\n", ip)
				c.Status(http.StatusTooManyRequests)
				return
			}
			s.ipConnections[ip]++
			s.ipMu.Unlock()

			defer func() {
				s.ipMu.Lock()
				s.ipConnections[ip]--
				if s.ipConnections[ip] == 0 {
					delete(s.ipConnections, ip)
				}
				s.ipMu.Unlock()
			}()

			m.HandleRequest(c.Writer, c.Request)
		})
		v2.GET("/ws_healthcheck", func(c *gin.Context) {
			c.String(http.StatusOK, "OK")
		})
	}

	srv := &http.Server{
		Addr:    bindTo,
		Handler: g,
	}

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("listen: %v\n", err)
		}
	}()

	go func() {
		select {
		case <-ctx.Done():
			defer func() { s.Done <- struct{}{} }()

			if err := srv.Shutdown(ctx); err != nil {
				log.Fatalf("server shutdown failed: %v", err)
			}
		}
	}()
}

func (s *HTTP) Broadcast(msg string) {
	err := s.m.Broadcast([]byte(msg))
	if err != nil {
		log.Printf("Broadcast error: %v\n", err)
	}
}
