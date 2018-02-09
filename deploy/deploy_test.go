package deploy_test

import (
	"context"
	"database/sql"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/schemalex/schemalex"
	"github.com/schemalex/schemalex/deploy"
	"github.com/stretchr/testify/assert"
)

const databaseName = `test_schemadeploy`

func TestDiff(t *testing.T) {
	var dsn = "root:@tcp(127.0.0.1:3306)/mysql"
	db, err := sql.Open("mysql", dsn)
	if !assert.NoError(t, err, `connecting to mysqld should succeed`) {
		return
	}
	defer db.Close()

	if _, err := db.Exec("CREATE DATABASE IF NOT EXISTS `" + databaseName + "`"); !assert.NoError(t, err, `CREATE DATABASE %s should succeed`, databaseName) {
		return
	}

	if _, err := db.Exec("USE `" + databaseName + "`"); !assert.NoError(t, err, `USE '%s' should succeed`, databaseName) {
		return
	}

	db.Exec("DROP TABLE `hoge`")
	db.Exec("DROP TABLE `fuga`")

	dir, err := ioutil.TempDir("", "schemadeploy")
	if !assert.NoError(t, err, `creating temporary directory should succeed`) {
		return
	}
	defer os.RemoveAll(dir)

	if !assert.NoError(t, os.Chdir(dir), `os.Chdir should succeed`) {
		return
	}

	if !assert.NoError(t, exec.Command("git", "init").Run(), `git init should succeed`) {
		return
	}

	if !assert.NoError(t, exec.Command("git", "config", "user.email", "hoge@example.com").Run(), `git config should succeed`) {
		return
	}

	if !assert.NoError(t, exec.Command("git", "config", "user.name", "hoge").Run(), `git config should succeed`) {
		return
	}

	schema, err := os.Create(filepath.Join(dir, "schema.sql"))
	if !assert.NoError(t, err, `os.Create should succeed`) {
		return
	}

	// first table
	if _, err := schema.WriteString("CREATE TABLE hoge ( `id` INTEGER NOT NULL, `c` VARCHAR(20) );\n"); !assert.NoError(t, err, `writing schema to file should succeed`) {
		return
	}
	schema.Sync()

	if !assert.NoError(t, exec.Command("git", "add", "schema.sql").Run(), `git add should succeed`) {
		return
	}

	if !assert.NoError(t, exec.Command("git", "commit", "-m", "initial commit").Run(), `git commit should succeed`) {
		return
	}

	// This is a silly hack, but we need to change the DSN from "mysql" or
	// whatever to "test"
	dsn = regexp.MustCompile(`/[^/]+$`).ReplaceAllString(dsn, `/`+databaseName)

	from, err := schemalex.NewSchemaSource(`mysql://` + dsn)
	if !assert.NoError(t, err, `creating 'from' source should succeed`) {
		return
	}
	to, err := schemalex.NewSchemaSource(`local-git://` + dir + `?file=schema.sql&commitish=HEAD`)
	if !assert.NoError(t, err, `creating 'to' source should succeed`) {
		return
	}
	if !assert.NoError(t, deploy.Diff(context.Background(), from, to), `deploy.Diff should succeed`) {
		return
	}

	// deployed
	if _, err := db.Exec("INSERT INTO `hoge` (`id`, `c`) VALUES (1, '2')"); !assert.NoError(t, err, `operations on new table should succeed`) {
		return
	}

	// second table
	if _, err := schema.WriteString("CREATE TABLE fuga ( `id` INTEGER NOT NULL, `c` VARCHAR(20) );\n"); !assert.NoError(t, err, `writing to schema file again should succeed`) {
		return
	}
	schema.Sync()

	if !assert.NoError(t, exec.Command("git", "add", "schema.sql").Run(), `git add should succeed`) {
		return
	}
	if !assert.NoError(t, exec.Command("git", "commit", "--author", "hoge <hoge@example.com>", "-m", "second commit").Run(), `git commit should succeed`) {
		return
	}

	if !assert.NoError(t, deploy.Diff(context.Background(), from, to), `deploy.Diff should succeed`) {
		return
	}

	if _, err := db.Exec("INSERT INTO `fuga` (`id`, `c`) VALUES (1, '2')"); !assert.NoError(t, err, `operations on new table should succeed`) {
		return
	}

	err = deploy.Diff(context.Background(), from, to)
	if !assert.True(t, deploy.IsIdenticalVersionsError(err), `deploy.Diff should return identical version error`) {
		return
	}
}
