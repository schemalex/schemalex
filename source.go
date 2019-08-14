package schemalex

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/schemalex/schemalex/internal/errors"
)

// SchemaSource is the interface used for objects that provide us with
// a MySQL database schema to work with.
type SchemaSource interface {
	// WriteSchema is responsible for doing whatever necessary to retrieve
	// the database schema and write to the given io.Writer
	WriteSchema(io.Writer) error
}

type readerSource struct {
	src io.Reader
}

type mysqlSource string

type localFileSource string

type localGitSource struct {
	dir       string
	file      string
	commitish string
}

// NewSchemaSource creates a SchemaSource based on the given URI.
// Currently "-" (for stdin), "local-git://...", "mysql://...", and
// "file://..." are supported. A string that does not match any of
// the above patterns and has no scheme part is treated as a local file.
func NewSchemaSource(uri string) (SchemaSource, error) {
	// "-" is a special source, denoting stdin.
	if uri == "-" {
		return NewReaderSource(os.Stdin), nil
	}

	if strings.HasPrefix(uri, "mysql://") {
		// Treat the argument as a DSN for mysql.
		// DSN is everything after "mysql://", so let's be lazy
		// and use everything after the second slash
		return NewMySQLSource(uri[8:]), nil
	}

	u, err := url.Parse(uri)
	if err != nil {
		return nil, errors.Wrap(err, `failed to parse uri`)
	}

	switch strings.ToLower(u.Scheme) {
	case "local-git":
		// local-git:///path/to/dir?file=foo&commitish=bar
		q := u.Query()
		return NewLocalGitSource(u.Path, q.Get("file"), q.Get("commitish")), nil
	case "file", "":
		// Eh, no remote host, please
		if u.Host != "" && u.Host != "localhost" {
			return nil, errors.Wrap(err, `remote hosts for file:// sources are not supported`)
		}
		return NewLocalFileSource(u.Path), nil
	}

	return nil, errors.New("invalid source")
}

// NewReaderSource creates a SchemaSource whose contents are read from the
// given io.Reader.
func NewReaderSource(src io.Reader) SchemaSource {
	return &readerSource{src: src}
}

// NewMySQLSource creates a SchemaSource whose contents are derived by
// accessing the specified MySQL instance.
//
// MySQL sources respect extra parameters "ssl-ca", "ssl-cert", and
// "ssl-secret" (which all should point to local file names) when
// the "tls" parameter is set to some boolean true value. In this
// case, we register the given tls configuration using those values
// automatically.
//
// Please note that the "tls" parameter MUST BE A BOOLEAN. Otherwise
// we expect that you have already registered your tls configuration
// manually, and that you gave us the name of that configuration
func NewMySQLSource(s string) SchemaSource {
	return mysqlSource(s)
}

// NewLocalFileSource creates a SchemaSource whose contents are derived from
// the given local file
func NewLocalFileSource(s string) SchemaSource {
	return localFileSource(s)
}

// NewLocalGitSource creates a SchemaSource whose contents are derived from
// the given file at the given commit ID in a git repository.
func NewLocalGitSource(gitDir, file, commitish string) SchemaSource {
	return &localGitSource{
		dir:       gitDir,
		file:      file,
		commitish: commitish,
	}
}

func (s *readerSource) WriteSchema(dst io.Writer) error {
	if _, err := io.Copy(dst, s.src); err != nil {
		return errors.Wrap(err, `failed to write schema to dst`)
	}
	return nil
}

// MySQLConfig creates a *mysql.Config struct from the given DSN.
func (s mysqlSource) MySQLConfig() (*mysql.Config, error) {
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

		// When comparing two mysql schemas against eachother, we will have
		// multiple calls to RegisterTLSConfig, and in that case we need
		// unique names for both.
		//
		// Here, we do the poor man's UUID, and create a unique name
		b := make([]byte, 16)
		rand.Reader.Read(b)
		b[6] = (b[6] & 0x0F) | 0x40
		b[8] = (b[8] &^ 0x40) | 0x80
		tlsName := fmt.Sprintf("custom-tls-%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])

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
	return cfg, nil
}

func (s mysqlSource) open() (*sql.DB, error) {
	// attempt to open connection to mysql
	cfg, err := s.MySQLConfig()
	if err != nil {
		return nil, errors.Wrap(err, `failed to create MySQL config from source spec`)
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

	return NewReaderSource(&buf).WriteSchema(dst)
}

func (s localGitSource) WriteSchema(dst io.Writer) error {
	var out bytes.Buffer
	cmd := exec.Command("git", "show", fmt.Sprintf("%s:%s", s.commitish, s.file))
	cmd.Stdout = &out
	cmd.Dir = s.dir

	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, `failed to run git command: %s`, cmd.Args)
	}

	return NewReaderSource(&out).WriteSchema(dst)
}
