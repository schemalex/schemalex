package model_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/schemalex/schemalex/format"
	"github.com/schemalex/schemalex/model"
	"github.com/stretchr/testify/assert"
)

func TestTableColumnNormalize(t *testing.T) {
	type testCase struct {
		before, after model.TableColumn
	}

	for _, tc := range []testCase{
		{
			// foo VARCHAR (255) NOT NULL
			before: model.NewTableColumn("foo").
				SetType(model.ColumnTypeVarChar).
				SetLength(model.NewLength("255")).
				SetNullState(model.NullStateNotNull),
			// foo VARCHAR (255) NOT NULL
			after: model.NewTableColumn("foo").
				SetType(model.ColumnTypeVarChar).
				SetLength(model.NewLength("255")).
				SetNullState(model.NullStateNotNull),
		},
		{
			// foo VARCHAR NULL
			before: model.NewTableColumn("foo").
				SetType(model.ColumnTypeVarChar).
				SetNullState(model.NullStateNull),
			// foo VARCHAR DEFAULT NULL
			after: model.NewTableColumn("foo").
				SetType(model.ColumnTypeVarChar).
				SetNullState(model.NullStateNone).
				SetDefault("NULL", false),
		},
		{
			// foo INTEGER NOT NULL,
			before: model.NewTableColumn("foo").
				SetType(model.ColumnTypeInteger).
				SetNullState(model.NullStateNotNull),
			// foo INT (11) NOT NULL,
			after: model.NewTableColumn("foo").
				SetType(model.ColumnTypeInt).
				SetLength(model.NewLength("11")).
				SetNullState(model.NullStateNotNull),
		},
		{
			// foo INTEGER UNSIGNED NULL DEFAULT 0,
			before: model.NewTableColumn("foo").
				SetType(model.ColumnTypeInteger).
				SetUnsigned(true).
				SetNullState(model.NullStateNull).
				SetDefault("0", false),
			// foo INT (10) UNSIGNED DEFAULT 0,
			after: model.NewTableColumn("foo").
				SetType(model.ColumnTypeInt).
				SetLength(model.NewLength("10")).
				SetUnsigned(true).
				SetNullState(model.NullStateNone).
				SetDefault("0", false),
		},
		{
			// foo bigint null default null,
			before: model.NewTableColumn("foo").
				SetType(model.ColumnTypeBigInt).
				SetNullState(model.NullStateNull).
				SetDefault("NULL", false),
			// foo BIGINT (20) DEFAULT NULL,
			after: model.NewTableColumn("foo").
				SetType(model.ColumnTypeBigInt).
				SetLength(model.NewLength("20")).
				SetNullState(model.NullStateNone).
				SetDefault("NULL", false),
		},
		{
			// foo DECIMAL,
			before: model.NewTableColumn("foo").
				SetType(model.ColumnTypeNumeric).
				SetNullState(model.NullStateNone),
			// foo DECIMAL (10,0) DEFAULT NULL,
			after: model.NewTableColumn("foo").
				SetType(model.ColumnTypeDecimal).
				SetLength(model.NewLength("10").SetDecimal("0")).
				SetNullState(model.NullStateNone).
				SetDefault("NULL", false),
		},
		{
			// foo TEXT,
			before: model.NewTableColumn("foo").
				SetType(model.ColumnTypeText),
			// foo TEXT,
			after: model.NewTableColumn("foo").
				SetType(model.ColumnTypeText),
		},
		{
			// foo BOOL,
			before: model.NewTableColumn("foo").
				SetType(model.ColumnTypeBool),
			// foo TINYINT(1) DEFAULT NULL,
			after: model.NewTableColumn("foo").
				SetType(model.ColumnTypeTinyInt).
				SetLength(model.NewLength("1")).
				SetNullState(model.NullStateNone).
				SetDefault("NULL", false),
		},
	} {
		var buf bytes.Buffer
		format.SQL(&buf, tc.before)
		beforeStr := buf.String()
		buf.Reset()
		format.SQL(&buf, tc.after)
		afterStr := buf.String()
		t.Run(fmt.Sprintf("from %s to %s", beforeStr, afterStr), func(t *testing.T) {
			norm, _ := tc.before.Normalize()
			if !assert.Equal(t, norm, tc.after, "Unexpected return value.") {
				buf.Reset()
				format.SQL(&buf, norm)
				normStr := buf.String()
				t.Logf("before: %s normlized: %s", beforeStr, normStr)
				t.Logf("after: %s", afterStr)
			}
		})
	}
}
