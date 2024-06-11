package oidc_broker

import (
	"crypto/rsa"
	"encoding/base64"
	"math/big"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Jwk represents a single JSON Web Key
type Jwk struct {
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	Kid string `json:"kid"`
}

// Jwks represents a JSON Web Key Set
type Jwks struct {
	Keys []Jwk `json:"keys"`
}

func (om *OidcBrokerManager) jwksHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"keys": om.jwks})
}

func publicKeyToJWK(pub *rsa.PublicKey, kid string) Jwk {
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())

	return Jwk{
		Kty: "RSA",
		N:   n,
		E:   e,
		Alg: "RS256",
		Use: "sig",
		Kid: kid,
	}
}
