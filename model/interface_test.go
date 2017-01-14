package model

import "testing"

func TestInterfaces(t *testing.T) {
	{
		var stmt Index
		stmt = &index{}
	}
	{
		var stmt TableColumn
		stmt= &tablecol{}
	}
}
