package config

import (
	"os"
)

var modes = map[string]bool{"SESSIONS": true,
	"OIDC_BROKER": true,
	"OIDC_SERVER": true}

type OidcConfig struct {
	EncryptionKey string
	PrivateKey    string
	PublicKey     string
	Providers     []ProviderConfig
}

type ProviderConfig struct {
	ProviderName string
	ProviderType string
	ProviderUrl  string
	ClientID     string `json:"clientId"`
	ClientSecret string
}

type Config struct {
	Mode           string
	KubeConfig     string
	SyncInterval   int
	CookieStoreKey string
	Oidc           OidcConfig
}

func (c *Config) ReadEnv() {
	c.KubeConfig = os.Getenv("KUBECONFIG")
}
