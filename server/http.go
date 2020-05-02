package server

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gopkg.in/olahol/melody.v1"
)

type HTTP struct {
	Done chan struct{}

	ctx context.Context
	m   *melody.Melody
}

func (s *HTTP) Start(ctx context.Context, bindTo string) {
	s.Done = make(chan struct{}, 1)
	s.ctx = ctx

	g := gin.Default()
	g.Use(cors.Default())

	m := melody.New()
	s.m = m

	m.HandleConnect(func(s *melody.Session) {
		log.Printf("Connection established.\tAddr:%s\tUser-Agent:%s\n", s.Request.RemoteAddr, s.Request.Header["User-Agent"])
	})

	m.HandleDisconnect(func(s *melody.Session) {
		log.Printf("Connection closed.\tAddr:%s\n", s.Request.RemoteAddr)
	})

	m.HandleError(func(s *melody.Session, e error) {
		log.Printf("Error occured.\tAddr:%s\tError:%#v\n", s.Request.RemoteAddr, e)
	})

	m.HandleSentMessage(func(s *melody.Session, b []byte) {
		log.Printf("Message sent.\tAddr:%s\n", s.Request.RemoteAddr)
	})

	v2 := g.Group("/v2")
	{
		v2.GET("/ws", func(c *gin.Context) {
			m.HandleRequest(c.Writer, c.Request)
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
