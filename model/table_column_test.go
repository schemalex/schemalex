package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTableColumnNormalize(t *testing.T) {
	type testCase struct {
		before, after *tablecol
	}

	for _, tc := range []testCase{
		{
			// foo VARCHAR (255) NOT NULL
			before: &tablecol{
				name:      "foo",
				typ:       ColumnTypeVarChar,
				length:    NewLength("255"),
				nullstate: NullStateNotNull,
			},
			// foo VARCHAR (255) NOT NULL
			after: &tablecol{
				name:      "foo",
				typ:       ColumnTypeVarChar,
				length:    NewLength("255"),
				nullstate: NullStateNotNull,
			},
		},
		{
			// foo VARCHAR NULL
			before: &tablecol{
				name:      "foo",
				typ:       ColumnTypeVarChar,
				nullstate: NullStateNull,
			},
			// foo VARCHAR DEFAULT NULL
			after: &tablecol{
				name:      "foo",
				typ:       ColumnTypeVarChar,
				nullstate: NullStateNone,
				defaultValue: defaultValue{
					Valid:  true,
					Value:  "NULL",
					Quoted: false,
				},
			},
		},
		{
			// foo INTEGER NOT NULL,
			before: &tablecol{
				name:      "foo",
				typ:       ColumnTypeInteger,
				nullstate: NullStateNotNull,
			},
			// foo INT (11) NOT NULL,
			after: &tablecol{
				name:      "foo",
				typ:       ColumnTypeInt,
				length:    NewLength("11"),
				nullstate: NullStateNotNull,
			},
		},
		{
			// foo INTEGER UNSIGNED NULL DEFAULT 0,
			before: &tablecol{
				name:      "foo",
				typ:       ColumnTypeInteger,
				unsigned:  true,
				nullstate: NullStateNull,
				defaultValue: defaultValue{
					Valid:  true,
					Value:  "0",
					Quoted: false,
				},
			},
			// foo INT (10) UNSIGNED DEFAULT 0,
			after: &tablecol{
				name:      "foo",
				typ:       ColumnTypeInt,
				length:    NewLength("10"),
				unsigned:  true,
				nullstate: NullStateNone,
				defaultValue: defaultValue{
					Valid:  true,
					Value:  "0",
					Quoted: false,
				},
			},
		},
		{
			// foo bigint null default null,
			before: &tablecol{
				name:      "foo",
				typ:       ColumnTypeBigInt,
				nullstate: NullStateNull,
				defaultValue: defaultValue{
					Valid:  true,
					Value:  "null",
					Quoted: false,
				},
			},
			// foo BIGINT (20) DEFAULT NULL,
			after: &tablecol{
				name:      "foo",
				typ:       ColumnTypeBigInt,
				length:    NewLength("20"),
				nullstate: NullStateNone,
				defaultValue: defaultValue{
					Valid:  true,
					Value:  "NULL",
					Quoted: false,
				},
			},
		},
		{
			// foo DECIMAL,
			before: &tablecol{
				name:      "foo",
				typ:       ColumnTypeNumeric,
				nullstate: NullStateNone,
			},
			// foo DECIMAL (10,0) DEFAULT NULL,
			after: &tablecol{
				name: "foo",
				typ:  ColumnTypeDecimal,
				length: func() Length {
					len := NewLength("10")
					len.SetDecimal("0")
					return len
				}(),
				nullstate: NullStateNone,
				defaultValue: defaultValue{
					Valid:  true,
					Value:  "NULL",
					Quoted: false,
				},
			},
		},
		{
			// foo TEXT,
			before: &tablecol{
				name: "foo",
				typ:  ColumnTypeText,
			},
			// foo TEXT,
			after: &tablecol{
				name: "foo",
				typ:  ColumnTypeText,
			},
		},
	} {
		assert.Equal(t, tc.before.Normalize(), tc.after, "Unexpected return value.")
	}
}
