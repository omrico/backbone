package config

import (
	"os"
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
	c.KubeConfig = os.Getenv("KUBECONFIG")
}
