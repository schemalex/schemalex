package deploy

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/schemalex/schemalex"
	"github.com/schemalex/schemalex/diff"
	"github.com/schemalex/schemalex/internal/errors"
)

type gitSource interface {
	schemalex.SchemaSource
	File() string
	Dir() string
	Commitish() string
}

type mysqlSource interface {
	schemalex.SchemaSource
	Open() (*sql.DB, error)
}

type errIdenticalVersions struct{}

func (e errIdenticalVersions) Error() string {
	return "identical versions"
}
func (e errIdenticalVersions) IsIdenticalVersions() bool {
	return true
}

type isIdenticalVersionsError interface {
	IsIdenticalVersions() bool
}

func IsIdenticalVersionsError(err error) bool {
	if err == nil {
		return false
	}

	if ive, ok := err.(isIdenticalVersionsError); ok {
		return ive.IsIdenticalVersions()
	}

	cerr := errors.Cause(err)
	if cerr == err {
		return false
	}

	return IsIdenticalVersionsError(cerr)
}

// Diff takes the two schema sources, creates a diff, and deploys the difference
// to the database source specified by the `from` parameter.
//
// If `to` is determined to be a git source, it will not only deploy the schema
// to the destination, but it will also record the currently deployed schema
// version in the designated table.
//
// The table to store the current deployed version must have at least the
// following columns:
//
//   version VARCHAR(40) NOT NULL
//
// If the deployed schema version and the yet-to-be deployed commit hash
// are equal, a special error is returned. You should use deploy.IsIdentialVersionsError
// to determine if the error means the schemas are identical
func Diff(ctx context.Context, from, to schemalex.SchemaSource) error {
	mysqlsrc, ok := from.(mysqlSource)
	if !ok {
		return errors.New(`'from' schema must be a valid mysql source`)
	}

	var hash string
	if gitsrc, ok := to.(gitSource); ok {
		// local-git implies that we have a git repository checked out somewhere
		// locally. make sure that we do...
		if _, err := os.Stat(filepath.Join(gitsrc.Dir(), ".git")); err != nil {
			return errors.Wrapf(err, `could not find .git under %s`, gitsrc.Dir())
		}

		v, err := localVersion(gitsrc)
		if err != nil {
			return errors.Wrap(err, `failed to determine local git SHA1 version to deploy`)
		}
		hash = v
	}

	db, err := mysqlsrc.Open()
	if err != nil {
		return errors.Wrap(err, `failed to open connection to database`)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, `failed to begin transaction`)
	}

	if hash != "" {
		if isIdenticalVersions(ctx, tx, hash) {
			return errIdenticalVersions{}
		}
	}

	if err := deployDiff(ctx, tx, from, to); err != nil {
		return errors.Wrap(err, `faild to deploy schema`)
	}

	if hash != "" {
		if err := deployVersion(ctx, tx, hash); err != nil {
			return errors.Wrap(err, `failed to store schema version`)
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, `failed to commit`)
	}
	return nil
}

func localVersion(gitsrc gitSource) (string, error) {
	var out bytes.Buffer
	cmd := exec.Command("git", "rev-parse", gitsrc.Commitish())
	cmd.Stdout = &out
	cmd.Dir = gitsrc.Dir()

	if err := cmd.Run(); err != nil {
		return "", errors.Wrapf(err, `failed to run git command: %s`, cmd.Args)
	}

	return strings.TrimSpace(out.String()), nil
}

func deployDiff(ctx context.Context, tx *sql.Tx, from, to schemalex.SchemaSource) error {
	var dst bytes.Buffer
	if err := diff.Sources(&dst, from, to, diff.WithTransaction(false)); err != nil {
		return errors.Wrap(err, `failed to generate diffs`)
	}

	scanner := bufio.NewScanner(&dst)
	scanner.Split(func(data []byte, atEOF bool) (int, []byte, error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.IndexByte(data, ';'); i >= 0 {
			return i + 1, data[0:i], nil
		}

		if atEOF {
			return len(data), data, nil
		}

		return 0, nil, nil
	})

	for scanner.Scan() {
		query := scanner.Text()
		if _, err := tx.Exec(query); err != nil {
			return errors.Wrapf(err, `failed to execute "%s"`, query)
		}
	}

	return nil
}

func deployVersion(ctx context.Context, tx *sql.Tx, hash string) error {
	if _, err := tx.Exec(`CREATE TABLE schemadeploy_version (version VARCHAR(40) NOT NULL)`); err != nil {
		return errors.Wrap(err, `failed to create schemadeploy_version table`)
	}

	if _, err := tx.Exec(`REPLACE INTO schemadeploy_version (version) VALUES (?)`, hash); err != nil {
		return errors.Wrap(err, `failed to insert new version`)
	}
	return nil
}

func isIdenticalVersions(ctx context.Context, tx *sql.Tx, hash string) bool {
	var remoteVersion string
	if err := tx.QueryRow("SELECT version FROM schemadeploy_version").Scan(&remoteVersion); err != nil {
		return false
	}

	return hash == remoteVersion
}
