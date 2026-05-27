package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gopkg.in/olahol/melody.v1"
)

type HTTP struct {
	Done chan struct{}

	ctx     context.Context
	m       *melody.Melody
	ipCount map[string]int
	mu      sync.Mutex
}

func getIP(s *melody.Session) string {
	if realIP := s.Request.Header.Get("X-Real-Ip"); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(s.Request.RemoteAddr)
	if err != nil {
		return s.Request.RemoteAddr
	}
	return host
}

func (s *HTTP) Start(ctx context.Context, bindTo string) {
	s.Done = make(chan struct{}, 1)
	s.ctx = ctx
	s.ipCount = make(map[string]int)

	g := gin.Default()
	g.Use(cors.Default())

	m := melody.New()
	s.m = m

	m.HandleConnect(func(sess *melody.Session) {
		ip := getIP(sess)
		s.mu.Lock()
		s.ipCount[ip]++
		s.mu.Unlock()
		log.Printf("Connection established.\tAddr:%s\tX-Real-Ip:%s\tUser-Agent:%s\n", sess.Request.RemoteAddr, sess.Request.Header["X-Real-Ip"], sess.Request.Header["User-Agent"])
	})

	m.HandleDisconnect(func(sess *melody.Session) {
		ip := getIP(sess)
		s.mu.Lock()
		s.ipCount[ip]--
		if s.ipCount[ip] <= 0 {
			delete(s.ipCount, ip)
		}
		s.mu.Unlock()
		log.Printf("Connection closed.\tAddr:%s\n", sess.Request.RemoteAddr)
	})

	m.HandleError(func(sess *melody.Session, e error) {
		log.Printf("Error occured.\tAddr:%s\tError:%#v\n", sess.Request.RemoteAddr, e)
	})

	m.HandleSentMessage(func(sess *melody.Session, b []byte) {
		log.Printf("Message sent.\tAddr:%s\n", sess.Request.RemoteAddr)
	})

	g.GET("/", func(c *gin.Context) {
		c.String(200, "Hello world!")
	})

	v2 := g.Group("/v2")
	{
		v2.GET("/ws", func(c *gin.Context) {
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
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.mu.Lock()
				for ip, count := range s.ipCount {
					if count >= 2 {
						fmt.Printf("[IPAddressStat] IP connections:\t%s\t%d\n", ip, count)
					}
				}
				s.mu.Unlock()
			case <-ctx.Done():
				return
			}
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
