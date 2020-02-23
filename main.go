package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/olahol/melody.v1"
	"log"
)

type Config struct {
	ApiKey string `envconfig:"api_key"`
}

func main() {
	var config Config
	err := envconfig.Process("", &config)
	if err != nil {
		log.Fatal(err)
	}
	if config.ApiKey == "" {
		log.Fatal("Please set API_KEY")
	}

	// Initialize frameworks
	r := gin.Default()
	r.Use(cors.Default())

	m := melody.New()

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

	// Define endpoints
	v2 := r.Group("/v2")
	{
		v2.GET("/ws", func(c *gin.Context) {
			m.HandleRequest(c.Writer, c.Request)
		})

		admin := v2.Group("/admin")
		{
			admin.POST("/broadcast", func(c *gin.Context) {
				key := c.DefaultQuery("api_key", "")
				if key != config.ApiKey {
					c.Status(400)
					return
				}
				data, _ := c.GetRawData()
				log.Printf("Message sending...\tMessage:%s\n", data)
				m.Broadcast([]byte(data))
			})
		}
	}

	// Run
	log.Println("Application started.")
	r.Run()
}
