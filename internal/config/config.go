package config

import (
	"os"
	"strconv"

	"github.com/omrico/backbone/internal/misc"

	"github.com/rs/xid"
)

var modes = map[string]bool{"SESSIONS": true,
	"OIDC_BROKER": true,
	"OIDC_SERVER": true}

type Config struct {
	Mode           string
	KubeConfig     string
	SyncInterval   int
	CookieStoreKey string
}

func (c *Config) ReadEnv() {
	logger := misc.GetLogger()
	c.KubeConfig = os.Getenv("KUBECONFIG")
	c.Mode = os.Getenv("MODE")
	mode := modes[c.Mode]
	if !mode {
		logger.Errorf("unknown mode or mode not provided")
		os.Exit(1)
	}

	syncInterval, ok := os.LookupEnv("SYNC_INTERVAL_SECONDS")
	if ok {
		val, err := strconv.Atoi(syncInterval)
		if err != nil {
			c.SyncInterval = 30
		} else {
			c.SyncInterval = val
		}
	} else {
		c.SyncInterval = 30
	}

	cookieStoreKey, ok := os.LookupEnv("COOKIE_STORE_KEY")
	if ok {
		c.CookieStoreKey = cookieStoreKey
	} else {
		logger.Info("generating random cookie store key")
		c.CookieStoreKey = xid.New().String()
	}
}
