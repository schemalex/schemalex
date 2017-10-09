package schemalex

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func generateCertificate(host string, certFile, secretFile io.Writer, isCA bool) error {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Schemalex Group"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	hosts := strings.Split(host, ",")
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	if isCA {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	pem.Encode(secretFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	return nil

}

func TestSchemaSource(t *testing.T) {
	certFile, err := ioutil.TempFile("", "schemalex-cert-")
	if !assert.NoError(t, err, "creating temporary file should succeed") {
		return
	}
	defer certFile.Close()
	defer os.Remove(certFile.Name())

	secretFile, err := ioutil.TempFile("", "schemalex-secret-")
	if !assert.NoError(t, err, "creating temporary file should succeed") {
		return
	}
	defer secretFile.Close()
	defer os.Remove(secretFile.Name())

	caCertFile, err := ioutil.TempFile("", "schemalex-ca-")
	if !assert.NoError(t, err, "creating temporary file should succeed") {
		return
	}
	defer caCertFile.Close()
	defer os.Remove(caCertFile.Name())

	if !assert.NoError(t, generateCertificate("schemalex.github.io", certFile, secretFile, false), "generating certificates should succeed") {
		return
	}
	if !assert.NoError(t, generateCertificate("schemalex.github.io", caCertFile, ioutil.Discard, true), "generating server CA should succeed") {
		return
	}

	type checker func(s SchemaSource) bool

	testcases := []struct {
		Input string
		Check []checker
		Error bool
	}{
		{
			Input: "/path/to/source.sql",
			Check: []checker{
				func(s SchemaSource) bool {
					lfs, ok := s.(localFileSource)
					if !assert.True(t, ok, `expected source to be a local file source, got %T`, s) {
						return false
					}
					if !assert.Equal(t, "/path/to/source.sql", string(lfs), "paths should match") {
						return false
					}
					return true
				},
			},
		},
		{
			Input: "file:///path/to/source.sql",
			Check: []checker{
				func(s SchemaSource) bool {
					lfs, ok := s.(localFileSource)
					if !assert.True(t, ok, `expected source to be a local file source, got %T`, s) {
						return false
					}
					if !assert.Equal(t, "/path/to/source.sql", string(lfs), "paths should match") {
						return false
					}
					return true
				},
			},
		},
		{
			Input: "-",
			Check: []checker{
				func(s SchemaSource) bool {
					_, ok := s.(*readerSource)
					if !assert.True(t, ok, `expected source to be reader source, got %T`, s) {
						return false
					}
					return true
				},
			},
		},
		{
			Input: "mysql://user:pass@tcp(1.2.3.4:9999)/dbname",
			Check: []checker{
				func(s SchemaSource) bool {
					_, ok := s.(mysqlSource)
					if !assert.True(t, ok, `expected source to be mysql source, got %T`, s) {
						return false
					}
					return true
				},
			},
		},
		{
			Input: "mysql://user:pass@tcp(1.2.3.4:9999)/dbname?tls=true",
			Check: []checker{
				func(s SchemaSource) bool {
					ms, ok := s.(mysqlSource)
					if !assert.True(t, ok, `expected source to be mysql source, got %T`, s) {
						return false
					}

					_, err := ms.MySQLConfig()
					if !assert.Error(t, err, "should error, because no tls configuration is provided") {
						return false
					}

					return true
				},
			},
		},
		{
			Input: fmt.Sprintf(
				"mysql://user:pass@tcp(1.2.3.4:9999)/dbname?tls=true&ssl-ca=%s&ssl-cert=%s&ssl-secret=%s",
				url.QueryEscape(caCertFile.Name()),
				url.QueryEscape(certFile.Name()),
				url.QueryEscape(secretFile.Name()),
			),
			Check: []checker{
				func(s SchemaSource) bool {
					ms, ok := s.(mysqlSource)
					if !assert.True(t, ok, `expected source to be mysql source, got %T`, s) {
						return false
					}

					cfg, err := ms.MySQLConfig()
					if !assert.NoError(t, err, "should be able to parse DSN") {
						return false
					}
					if !assert.True(t, strings.HasPrefix(cfg.TLSConfig, "custom-tls"), "TLSConfig should be enabled") {
						return false
					}

					return true
				},
			},
		},
		{
			Input: "local-git:///path/to/dir?file=foo/baz.sql&commitish=deadbeaf",
			Check: []checker{
				func(s SchemaSource) bool {
					lgs, ok := s.(*localGitSource)
					if !assert.True(t, ok, `expected source to be local git source, got %T`, s) {
						return false
					}

					if !assert.Equal(t, "/path/to/dir", lgs.dir, "directory should match") {
						return false
					}
					if !assert.Equal(t, "foo/baz.sql", lgs.file, "file should match") {
						return false
					}
					if !assert.Equal(t, "deadbeaf", lgs.commitish, "commit ID should match") {
						return false
					}
					return true
				},
			},
		},
		{Input: "https://github.com/schemalex/schemalex", Error: true},
	}

	for _, c := range testcases {
		t.Run(fmt.Sprintf("Parse %s", strconv.Quote(c.Input)), func(t *testing.T) {
			s, err := NewSchemaSource(c.Input)
			if c.Error {
				assert.Error(t, err, "expected '%s' to result in an error")

				// if we expected an error, we have nothing more to do in this
				// particular test case regardless of the previous assert
				return

			}

			if !assert.NoError(t, err, "expected '%s' to successfully parse") {
				return
			}

			if len(c.Check) == 0 {
				return
			}

			for _, check := range c.Check {
				if !check(s) {
					return
				}
			}
		})
	}
}
