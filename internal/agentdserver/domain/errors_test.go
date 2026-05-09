package domain

import (
	"fmt"
	"testing"
)

func TestErrorCodeForContractAndProviderFailures(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		err  error
		want ErrorCode
	}{
		"invalid contract schema": {
			err:  ErrInvalidContractSchema,
			want: ErrorCodeContractSchemaInvalid,
		},
		"contract input invalid": {
			err:  ErrContractInputInvalid,
			want: ErrorCodeContractInputInvalid,
		},
		"contract output invalid": {
			err:  ErrContractOutputInvalid,
			want: ErrorCodeContractOutputInvalid,
		},
		"provider unavailable": {
			err:  ErrProviderUnavailable,
			want: ErrorCodeProviderUnavailable,
		},
		"provider request failed": {
			err:  ErrProviderRequestFailed,
			want: ErrorCodeProviderRequestFailed,
		},
		"wrapped provider error": {
			err:  fmt.Errorf("execute model request: %w", ErrProviderRequestFailed),
			want: ErrorCodeProviderRequestFailed,
		},
		"unknown error": {
			err:  fmt.Errorf("unexpected"),
			want: ErrorCodeInternal,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := ErrorCodeFor(tt.err); got != tt.want {
				t.Fatalf("ErrorCodeFor(%v): got %q want %q", tt.err, got, tt.want)
			}
		})
	}
}

func TestErrorCodeForNil(t *testing.T) {
	t.Parallel()

	if got := ErrorCodeFor(nil); got != "" {
		t.Fatalf("nil error code: got %q", got)
	}
}
