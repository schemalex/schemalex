package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/pkg/errors"
	"github.com/schemalex/schemalex"
	"github.com/schemalex/schemalex/deploy"
)

var version string

func main() {
	if err := _main(); err != nil {
		log.Printf("%s", err)
		os.Exit(1)
	}
}

func _main() error {
	var showVersion bool

	flag.Usage = func() {
		fmt.Printf(`schemadeploy version %s

schemadeploy -version
schemadeploy [options...] from to

-v            Print out the version and exit

"from" must be a valid "mysql" source.
"to" may be a file path, or a URI.
Special URI schemes "mysql" and "local-git" are supported on top of
"file". If the special path "-" is used, it is treated as stdin.

If "local-git" is used as "to", then the deployed version is always
recorded in the database

Examples:

* Deploy a local file
  schemadeploy \
	  "mysql://user:password@tcp(host:port)dbname?option=value" /path/to/file

* Deploy a file in local git repository 
  schemalint  \
	  "mysql://user:password@tcp(host:port)dbname?option=value" \
		local-git:///path/to/repo?file=foo.sql&commitish=deadbeaf

`, version)
	}
	flag.BoolVar(&showVersion, "v", false, "")
	flag.Parse()

	if showVersion {
		if version == "" {
			version = "(custom build)"
		}

		fmt.Printf(
			"schemadeploy version %s, built with %s (%s/%s)\n",
			version,
			runtime.Version(),
			runtime.GOOS,
			runtime.GOARCH,
		)
		return nil
	}

	if flag.NArg() != 2 {
		flag.Usage()
		return errors.New("wrong number of arguments")
	}

	from, err := schemalex.NewSchemaSource(flag.Arg(0))
	if err != nil {
		return errors.Wrap(err, `failed to create schema source for "from"`)
	}

	to, err := schemalex.NewSchemaSource(flag.Arg(1))
	if err != nil {
		return errors.Wrap(err, `failed to create schema source for "to"`)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := deploy.Diff(ctx, from, to); err != nil {
		if deploy.IsIdenticalVersionsError(err) {
			log.Printf("An identical schema version is already deployed")
			return nil
		}
		return errors.Wrap(err, `failed to deploy schema`)
	}

	return nil
}
