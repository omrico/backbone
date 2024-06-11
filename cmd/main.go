package main

import (
	"sync"

	"github.com/omrico/backbone/internal/config"
	"github.com/omrico/backbone/internal/k8s"
	logging "github.com/omrico/backbone/internal/misc"
	oidcbroker "github.com/omrico/backbone/internal/oidc-broker"
	"github.com/omrico/backbone/internal/sessions"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func main() {
	// init logger
	logger := logging.GetLogger()
	defer logging.GracefulShutdown()
	logger.Info("Initializing...")

	// init config, read env vars
	logger.Info("Reading env vars")
	cfg := &config.Config{}
	cfg.ReadEnv()

	c := &k8s.Client{
		Cfg: cfg,
	}
	c.NewClient()

	// use a waitgroup to prevent execution of some methods before config is ready
	var wg sync.WaitGroup
	wg.Add(1)

	// read the Backbone Config CR and load it to memory.
	// Also define the handler for Config changes event
	c.ConfigWithWatcher(cfg, &wg)

	// start the timed sync every X seconds
	c.StartSync(&wg)

	// middlewares and handlers
	r := gin.Default()
	r.Use(func(c *gin.Context) {
		// Generate a unique UUID for the request
		requestID := uuid.New().String()

		// Set the request ID in the context for later use
		c.Set("request-id", requestID)

		// Pass control to the next handler
		c.Next()
	})

	var sm sessions.SessionManager
	var om oidcbroker.OidcBrokerManager

	switch cfg.Mode {
	case "SESSIONS":
		sm = sessions.SessionManager{SyncClient: c, Cfg: cfg}
		sm.Init(r, &wg)
	case "OIDC_BROKER":
		om = oidcbroker.OidcBrokerManager{SyncClient: c, Cfg: cfg}
		om.Init(r, &wg)
	}

	logger.Info("Initializing... done")

	// start the server
	err := r.Run()
	if err != nil {
		logger.Fatalf("failed starting app: %s", err.Error())
	}
}
