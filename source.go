package schemalex

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/schemalex/schemalex/internal/errors"
)

type SchemaSource interface {
	WriteSchema(io.Writer) error
}

type mysqlSource string

type localFileSource string

func NewSchemaSource(uri string) (SchemaSource, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, errors.Wrap(err, `failed to parse uri`)
	}

	switch strings.ToLower(u.Scheme) {
	case "mysql":
		// Treat the argument as a DSN for mysql.
		// DSN is everything after "mysql://", so let's be lazy
		// and use everything after the second slash
		return mysqlSource(uri[8:]), nil
	case "file", "":
		// Eh, no remote host, please
		if u.Host != "" && u.Host != "localhost" {
			return nil, errors.Wrap(err, `remote hosts for file:// sources are not supported`)
		}

		return localFileSource(u.Path), nil
	}

	return nil, errors.New("invalid source")
}

func (s mysqlSource) open() (*sql.DB, error) {
	// attempt to open connection to mysql
	cfg, err := mysql.ParseDSN(string(s))
	if err != nil {
		return nil, errors.Wrap(err, `failed to parse DSN`)
	}

	// because _I_ need support for tls, I'm going to handle setting up
	// the tls stuff, by using
	// tls=true&ssl-ca=file=...&ssl-cert=...&ssql-secret=...
	if v, err := strconv.ParseBool(cfg.TLSConfig); err == nil && v {
		sslCa := cfg.Params["ssl-ca"]
		sslCert := cfg.Params["ssl-cert"]
		sslSecret := cfg.Params["ssl-secret"]
		if sslCa == "" || sslCert == "" || sslSecret == "" {
			return nil, errors.New(`to enable tls, you must provide ssl-ca, ssl-cert, and ssl-secret parameters to the DSN`)
		}

		tlsName := "custom-tls"
		rootCertPool := x509.NewCertPool()
		pem, err := ioutil.ReadFile(sslCa)
		if err != nil {
			return nil, errors.Wrap(err, `failed to read ssl-ca file`)
		}
		if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
			return nil, errors.New(`failed to append ssl-ca PEM to cert pool`)
		}
		certs, err := tls.LoadX509KeyPair(sslCert, sslSecret)
		if err != nil {
			return nil, errors.Wrap(err, `failed to load X509 key pair`)
		}
		mysql.RegisterTLSConfig(tlsName, &tls.Config{
			RootCAs:      rootCertPool,
			Certificates: []tls.Certificate{certs},
		})
		cfg.TLSConfig = tlsName
	}

	return sql.Open("mysql", cfg.FormatDSN())
}

func (s localFileSource) WriteSchema(dst io.Writer) error {
	f, err := os.Open(string(s))
	if err != nil {
		return errors.Wrapf(err, `failed to open local file %s`, s)
	}
	defer f.Close()

	if _, err := io.Copy(dst, f); err != nil {
		return errors.Wrap(err, `failed to copy file contents to dst`)
	}
	return nil
}

func (s mysqlSource) WriteSchema(dst io.Writer) error {
	db, err := s.open()
	if err != nil {
		return errors.Wrap(err, `failed to open connection to database`)
	}
	defer db.Close()

	tableRows, err := db.Query("SHOW TABLES")
	if err != nil {
		return errors.Wrap(err, `failed to execute 'SHOW TABLES'`)
	}
	defer tableRows.Close()

	var table string
	var tableSchema string
	var buf bytes.Buffer
	for tableRows.Next() {
		if err = tableRows.Scan(&table); err != nil {
			return errors.Wrap(err, `failed to scan tables`)
		}

		if err = db.QueryRow("SHOW CREATE TABLE `"+table+"`").Scan(&table, &tableSchema); err != nil {
			return errors.Wrapf(err, `failed to execute 'SHOW CREATE TABLE "%s"'`, table)
		}
		if buf.Len() > 0 {
			buf.WriteString("\n\n")
		}
		// TODO remove dynamic info. ex) AUTO_INCREMENT,PARTITION
		buf.WriteString(tableSchema)
		buf.WriteByte(';')
	}
	if _, err := buf.WriteTo(dst); err != nil {
		return errors.Wrap(err, `failed to write schema to dst`)
	}
	return nil
}
