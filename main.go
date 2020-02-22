package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/olahol/melody.v1"
)

type Config struct {
	ApiKey string `envconfig:"api_key"`
}

func main() {
	var config Config
	err := envconfig.Process("", &config)
	if err != nil {
		return
	}

	r := gin.Default()
	r.Use(cors.Default())

	m := melody.New()

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
				m.Broadcast([]byte(data))
			})
		}
	}

	r.Run()
}
