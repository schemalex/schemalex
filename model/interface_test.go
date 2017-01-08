package model

import "testing"

func TestInterfaces(t *testing.T) {
	{
		var stmt Index = &index{}
		_ = stmt
	}
	{
		var stmt TableColumn = &tablecol{}
		_ = stmt
	}
}
