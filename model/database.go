package model

import (
	"bytes"
	"io"

	"github.com/schemalex/schemalex/internal/util"
)

func NewDatabase(n string) Database {
	return &database{
		name: n,
	}
}

func (d *database) ID() string {
	return "database#" + d.name
}

func (d *database) Name() string {
	return d.name
}

func (d *database) IsIfNotExists() bool {
	return d.ifnotexists
}

func (d *database) SetIfNotExists(v bool) {
	d.ifnotexists = v
}

func (d *database) WriteTo(dst io.Writer) (int64, error) {
	var buf bytes.Buffer
	buf.WriteString("CREATE DATABASE")
	if d.IsIfNotExists() {
		buf.WriteString(" IF NOT EXISTS")
	}
	buf.WriteByte(' ')
	buf.WriteString(util.Backquote(d.Name()))
	buf.WriteByte(';')

	return buf.WriteTo(dst)
}
