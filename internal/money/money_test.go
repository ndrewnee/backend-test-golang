package money

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAmount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{name: "valid integer", raw: "100", want: "100.00"},
		{name: "valid cents", raw: "100.50", want: "100.50"},
		{name: "empty", raw: "", wantErr: true},
		{name: "zero", raw: "0", wantErr: true},
		{name: "negative", raw: "-1", wantErr: true},
		{name: "too many decimals", raw: "1.001", wantErr: true},
		{name: "not decimal", raw: "abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseAmount(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, Format(got))
		})
	}
}
