package transfer

import (
	"testing"

	"github.com/shopspring/decimal"
)

func Test_splitChange(t *testing.T) {
	newDecimal := func(s string) decimal.Decimal {
		d, _ := decimal.NewFromString(s)
		return d
	}

	type args struct {
		amount decimal.Decimal
		n      int
	}
	tests := []struct {
		name string
		args args
		want []decimal.Decimal
	}{
		{
			name: "zero",
			args: args{
				amount: newDecimal("0"),
				n:      2,
			},
			want: nil,
		},
		{
			name: "split 1",
			args: args{
				amount: newDecimal("1"),
				n:      1,
			},
			want: []decimal.Decimal{
				newDecimal("1"),
			},
		},
		{
			name: "split 2",
			args: args{
				amount: newDecimal("2"),
				n:      2,
			},
			want: []decimal.Decimal{
				newDecimal("1"),
				newDecimal("1"),
			},
		},
		{
			name: "split small",
			args: args{
				amount: newDecimal("0.00000001"),
				n:      2,
			},
			want: []decimal.Decimal{
				newDecimal("0.00000001"),
			},
		},
		{
			name: "split small 2",
			args: args{
				amount: newDecimal("0.00000002"),
				n:      2,
			},
			want: []decimal.Decimal{
				newDecimal("0.00000001"),
				newDecimal("0.00000001"),
			},
		},
		{
			name: "split small 3",
			args: args{
				amount: newDecimal("0.00000002"),
				n:      3,
			},
			want: []decimal.Decimal{
				newDecimal("0.00000001"),
				newDecimal("0.00000001"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := splitChange(tt.args.amount, tt.args.n); !equalDecimalSlice(got, tt.want) {
				t.Errorf("splitChange() = %v, want %v", got, tt.want)
			}
		})
	}
}

func equalDecimalSlice(a, b []decimal.Decimal) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if !v.Equal(b[i]) {
			return false
		}
	}

	return true
}
