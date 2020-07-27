package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/inverse-inc/packetfence/go/log"
	"github.com/inverse-inc/packetfence/go/sharedutils"
	"github.com/jcuga/golongpoll"
)

const defaultPollTimeout = 30 * time.Second
const maxSessionIdleTime = 5 * time.Minute
const LONG_POLL_CONTEXT_KEY = "GLP-GIN-MIDDLEWARE"

func makeGinServer() *gin.Engine {
	r := gin.Default()
	r.Use(longPollMiddleware())
	r.GET("/profile/:node_id", handleGetProfile)
	r.GET("/peer/:node_id", handleGetPeer)
	r.GET("/events/:k", handleGetEvents)
	r.POST("/events/:k", handlePostEvents)
	return r
}

func renderError(c *gin.Context, code int, err error) {
	renderErrors(c, code, []error{err})
}

func renderErrors(c *gin.Context, code int, errs []error) {
	strErrs := []string{}
	for _, err := range errs {
		log.LoggerWContext(c).Error("Got the following error while processing the request: " + err.Error())
		strErrs = append(strErrs, err.Error())
	}
	c.JSON(code, gin.H{"errors": strErrs})
}

func longPollFromContext(c *gin.Context) *golongpoll.LongpollManager {
	if v, ok := c.Get(LONG_POLL_CONTEXT_KEY); ok {
		return v.(*golongpoll.LongpollManager)
	} else {
		return nil
	}
}

func longPollMiddleware() gin.HandlerFunc {
	pubsub, err := golongpoll.StartLongpoll(golongpoll.Options{
		LoggingEnabled:     (sharedutils.EnvOrDefault("LOG_LEVEL", "") == "debug"),
		MaxEventBufferSize: 1000,
		// Events stay for up to 5 minutes
		EventTimeToLiveSeconds: int(maxSessionIdleTime / time.Second),
	})
	sharedutils.CheckError(err)

	return func(c *gin.Context) {
		c.Set(LONG_POLL_CONTEXT_KEY, pubsub)
		c.Next()
	}
}

func main() {
	r := makeGinServer()
	r.Run()
}
