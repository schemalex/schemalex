package model

import "testing"

func TestInterfaces(t *testing.T) {
	{
		var stmt Index
		stmt = &index{}
		_ = stmt
	}
	{
		var stmt TableColumn
		stmt = &tablecol{}
		_ = stmt
	}
}
