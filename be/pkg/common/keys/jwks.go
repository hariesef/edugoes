package keys

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"sync"

	"github.com/google/uuid"
	jwk "github.com/lestrrat-go/jwx/v2/jwk"
)

// manager lazily generates and holds a platform RSA key and JWKS for signing id_tokens.
// For a PoC we keep it in-memory. In production, persist and allow rotation.
var (
	once       sync.Once
	platformJWKS jwk.Set
	platformKid  string
	platformKey  *rsa.PrivateKey
)

// Init ensures the JWKS is available.
func Init() error {
	var initErr error
	once.Do(func() {
		// Try to read a KID from env, else random UUID.
		kid := os.Getenv("PLATFORM_KID")
		if kid == "" {
			kid = uuid.NewString()
		}
		platformKid = kid

		// Prefer loading private key from environment (CI/CD-provided)
		var key *rsa.PrivateKey
		if b64 := os.Getenv("PLATFORM_PRIVATE_KEY_B64"); b64 != "" {
			der, err := base64.StdEncoding.DecodeString(b64)
			if err == nil {
				if block, _ := pem.Decode(der); block != nil {
					if k, err2 := x509.ParsePKCS1PrivateKey(block.Bytes); err2 == nil {
						key = k
					} else if pkcs8, err3 := x509.ParsePKCS8PrivateKey(block.Bytes); err3 == nil {
						if rk, ok := pkcs8.(*rsa.PrivateKey); ok {
							key = rk
						}
					}
				}
			}
		}
		if key == nil {
			if pemStr := os.Getenv("PLATFORM_PRIVATE_KEY_PEM"); pemStr != "" {
				if block, _ := pem.Decode([]byte(pemStr)); block != nil {
					if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
						key = k
					} else if pkcs8, err2 := x509.ParsePKCS8PrivateKey(block.Bytes); err2 == nil {
						if rk, ok := pkcs8.(*rsa.PrivateKey); ok {
							key = rk
						}
					}
				}
			}
		}
		// Fallback: generate a 2048-bit RSA key for dev.
		if key == nil {
			gen, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				initErr = err
				return
			}
			key = gen
			// Print helpers so the operator can capture and persist the key.
			block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(gen)}
			pemBytes := pem.EncodeToMemory(block)
			b64 := base64.StdEncoding.EncodeToString(pemBytes)
			fmt.Println("[keys] Generated ephemeral RSA key (dev mode). To persist, set one of:")
			fmt.Printf("export PLATFORM_PRIVATE_KEY_PEM='%s'\n", string(pemBytes))
			fmt.Printf("export PLATFORM_PRIVATE_KEY_B64='%s'\n", b64)
			fmt.Printf("export PLATFORM_KID='%s'\n", kid)
		}
		platformKey = key

		jwkKey, err := jwk.FromRaw(&key.PublicKey)
		if err != nil {
			initErr = err
			return
		}
		_ = jwkKey.Set("kid", kid)
		_ = jwkKey.Set("alg", "RS256")
		_ = jwkKey.Set("use", "sig")
		_ = jwkKey.Set("kty", "RSA")

		set := jwk.NewSet()
		set.AddKey(jwkKey)
		platformJWKS = set
	})
	return initErr
}

// JWKSJSON returns the JWKS as JSON bytes.
func JWKSJSON() ([]byte, error) {
	if err := Init(); err != nil {
		return nil, err
	}
	return json.Marshal(platformJWKS)
}

// PrivateKey returns the platform signing key (PoC use).
func PrivateKey() *rsa.PrivateKey {
	return platformKey
}

// Kid returns current key id.
func Kid() string { return platformKid }
