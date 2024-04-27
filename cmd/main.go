package main

import (
	"net/http"

	"github.com/omrico/backbone/internal/auth"
	"github.com/omrico/backbone/internal/config"
	"github.com/omrico/backbone/internal/k8s"
	logging "github.com/omrico/backbone/internal/misc"
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

	logger.Infof("Starting k8s sync client. Refresh will happen every %d seconds", cfg.SyncInterval)
	c := &k8s.Client{
		Cfg: cfg,
	}
	c.StartSync()

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

	sm := sessions.SessionManager{SyncClient: c, Cfg: cfg}
	sm.Init(r)

	r.GET("/main/ping", sm.SessionMiddleware(), func(c *gin.Context) {
		userAuth, err := auth.BuildAuthFromCtx(c)
		if err != nil {
			logger.Warnf("failed extracting roles from cookie: %s", err.Error())
		}
		c.JSON(http.StatusOK, gin.H{
			"username": userAuth.Username,
			"roles":    userAuth.RolesAndPerms,
		})
	})

	logger.Info("Initializing... done")
	//user.AddHandlers(r)

	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
