package key

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/ssh"
)

// Algorithm constants
const (
	RSA2048 = "rsa2048"
	RSA4096 = "rsa4096"
	P256    = "p256"
	P384    = "p384"
	P521    = "p521"
	ED25519 = "ed25519"
)

// KeyGenerator interface
type KeyGenerator interface {
	Generate(name, email string) (privateKey, publicKey string, err error)
}

// RSA2048Generator implements KeyGenerator for RSA 2048-bit
type RSA2048Generator struct{}

func (g *RSA2048Generator) Generate(name, email string) (string, string, error) {
	return generateRSA(2048, name, email)
}

// RSA4096Generator implements KeyGenerator for RSA 4096-bit
type RSA4096Generator struct{}

func (g *RSA4096Generator) Generate(name, email string) (string, string, error) {
	return generateRSA(4096, name, email)
}

// P256Generator implements KeyGenerator for ECDSA P-256
type P256Generator struct{}

func (g *P256Generator) Generate(name, email string) (string, string, error) {
	return generateECDSA(elliptic.P256(), name, email)
}

// P384Generator implements KeyGenerator for ECDSA P-384
type P384Generator struct{}

func (g *P384Generator) Generate(name, email string) (string, string, error) {
	return generateECDSA(elliptic.P384(), name, email)
}

// P521Generator implements KeyGenerator for ECDSA P-521
type P521Generator struct{}

func (g *P521Generator) Generate(name, email string) (string, string, error) {
	return generateECDSA(elliptic.P521(), name, email)
}

// ED25519Generator implements KeyGenerator for Ed25519
type ED25519Generator struct{}

func (g *ED25519Generator) Generate(name, email string) (string, string, error) {
	return generateEd25519(name, email)
}

// Global map of generators
var generators = map[string]KeyGenerator{
	RSA2048: &RSA2048Generator{},
	RSA4096: &RSA4096Generator{},
	P256:    &P256Generator{},
	P384:    &P384Generator{},
	P521:    &P521Generator{},
	ED25519: &ED25519Generator{},
}

// GetKeyGenerator returns the KeyGenerator for the given algorithm name
func GetKeyGenerator(algo string) (KeyGenerator, error) {
	gen, ok := generators[algo]
	if !ok {
		return nil, fmt.Errorf("unsupported algorithm: %s", algo)
	}
	return gen, nil
}

// Helper functions

func generateRSA(bits int, name, email string) (string, string, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return "", "", err
	}

	der := x509.MarshalPKCS1PrivateKey(privKey)
	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: der,
	})

	pubKey, err := ssh.NewPublicKey(&privKey.PublicKey)
	if err != nil {
		return "", "", err
	}

	pubKeyString := string(ssh.MarshalAuthorizedKey(pubKey))
	// ssh.MarshalAuthorizedKey ends with newline, trim it to append comment
	if len(pubKeyString) > 0 && pubKeyString[len(pubKeyString)-1] == '\n' {
		pubKeyString = pubKeyString[:len(pubKeyString)-1]
	}
	pubKeyString += fmt.Sprintf(" %s@%s", name, email)

	return string(privKeyPEM), pubKeyString, nil
}

func generateECDSA(curve elliptic.Curve, name, email string) (string, string, error) {
	privKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return "", "", err
	}

	der, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		// Fallback to PKCS#8 if MarshalECPrivateKey is not supported for the curve
		der, err = x509.MarshalPKCS8PrivateKey(privKey)
		if err != nil {
			return "", "", err
		}
		privKeyPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: der,
		})

		pubKey, err := ssh.NewPublicKey(&privKey.PublicKey)
		if err != nil {
			return "", "", err
		}

		pubKeyString := string(ssh.MarshalAuthorizedKey(pubKey))
		if len(pubKeyString) > 0 && pubKeyString[len(pubKeyString)-1] == '\n' {
			pubKeyString = pubKeyString[:len(pubKeyString)-1]
		}
		pubKeyString += fmt.Sprintf(" %s@%s", name, email)
		return string(privKeyPEM), pubKeyString, nil
	}

	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: der,
	})

	pubKey, err := ssh.NewPublicKey(&privKey.PublicKey)
	if err != nil {
		return "", "", err
	}

	pubKeyString := string(ssh.MarshalAuthorizedKey(pubKey))
	if len(pubKeyString) > 0 && pubKeyString[len(pubKeyString)-1] == '\n' {
		pubKeyString = pubKeyString[:len(pubKeyString)-1]
	}
	pubKeyString += fmt.Sprintf(" %s@%s", name, email)

	return string(privKeyPEM), pubKeyString, nil
}

func generateEd25519(name, email string) (string, string, error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}

	der, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return "", "", err
	}

	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: der,
	})

	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return "", "", err
	}

	pubKeyString := string(ssh.MarshalAuthorizedKey(sshPubKey))
	if len(pubKeyString) > 0 && pubKeyString[len(pubKeyString)-1] == '\n' {
		pubKeyString = pubKeyString[:len(pubKeyString)-1]
	}
	pubKeyString += fmt.Sprintf(" %s@%s", name, email)

	return string(privKeyPEM), pubKeyString, nil
}
