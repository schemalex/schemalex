package model_test

import (
	"testing"

	"github.com/schemalex/schemalex/model"
)

func TestStatement(t *testing.T) {
	var stmts model.Stmts

	stmts = append(stmts, model.NewDatabase("test"))
	stmts = append(stmts, model.NewTable("test"))
	stmts = append(stmts, model.NewTableColumn("test"))
	stmts = append(stmts, model.NewIndex(model.IndexKindPrimaryKey))
}
