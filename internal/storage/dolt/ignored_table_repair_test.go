package dolt

import "testing"

func TestIsIgnoredTableCorruptionError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "read many values checksum",
			err:  errString("writeCommitParentClosure: ReadManyValues: checksum error"),
			want: true,
		},
		{
			name: "plain checksum is ignored",
			err:  errString("checksum error"),
			want: false,
		},
		{
			name: "non corruption error",
			err:  errString("table not found"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isIgnoredTableCorruptionError(tt.err); got != tt.want {
				t.Fatalf("isIgnoredTableCorruptionError() = %v, want %v", got, tt.want)
			}
		})
	}
}

type errString string

func (e errString) Error() string { return string(e) }
