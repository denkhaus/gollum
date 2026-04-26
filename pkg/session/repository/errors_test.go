package repository

import (
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
)

func TestSessionError_Error(t *testing.T) {
	tests := []struct {
		name      string
		sessionID uuid.UUID
		op        string
		err       error
		want      string
	}{
		{
			name:      "with valid session ID",
			sessionID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
			op:        "Create",
			err:       errors.New("database error"),
			want:      "session 123e4567-e89b-12d3-a456-426614174000: Create failed: database error",
		},
		{
			name:      "with zero UUID",
			sessionID: uuid.Nil,
			op:        "Create",
			err:       errors.New("database error"),
			want:      "session: Create failed: database error",
		},
		{
			name:      "with empty operation",
			sessionID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
			op:        "",
			err:       errors.New("some error"),
			want:      "session 123e4567-e89b-12d3-a456-426614174000:  failed: some error",
		},
		{
			name:      "with nil error",
			sessionID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
			op:        "Update",
			err:       nil,
			want:      "session 123e4567-e89b-12d3-a456-426614174000: Update failed: <nil>",
		},
		{
			name:      "with zero UUID and nil error",
			sessionID: uuid.Nil,
			op:        "Delete",
			err:       nil,
			want:      "session: Delete failed: <nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &SessionError{
				SessionID: tt.sessionID,
				Op:        tt.op,
				Err:       tt.err,
			}
			got := e.Error()
			if got != tt.want {
				t.Errorf("SessionError.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSessionError_Unwrap(t *testing.T) {
	tests := []struct {
		name         string
		sessionID    uuid.UUID
		op           string
		err          error
		checkNil     bool
		checkWrapped bool
	}{
		{
			name:      "with valid error",
			sessionID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
			op:        "Create",
			err:       errors.New("underlying error"),
		},
		{
			name:      "with nil error",
			sessionID: uuid.Nil,
			op:        "Update",
			err:       nil,
			checkNil:  true,
		},
		{
			name:         "with wrapped error",
			sessionID:    uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
			op:          "Delete",
			err:          fmt.Errorf("wrapped: %w", ErrSessionNotFound),
			checkWrapped: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &SessionError{
				SessionID: tt.sessionID,
				Op:        tt.op,
				Err:       tt.err,
			}
			got := e.Unwrap()

			if tt.checkNil {
				if got != tt.err {
					t.Errorf("SessionError.Unwrap() = %v, want %v", got, tt.err)
				}
			} else if tt.checkWrapped {
				if !errors.Is(got, ErrSessionNotFound) {
					t.Errorf("SessionError.Unwrap() should wrap ErrSessionNotFound, got %v", got)
				}
			} else {
				if got == nil || got.Error() != tt.err.Error() {
					t.Errorf("SessionError.Unwrap() = %v, want %v", got, tt.err)
				}
			}
		})
	}
}

func TestSessionError_ErrorUnwrapIntegration(t *testing.T) {
	// Test that errors.Is and errors.As work correctly with SessionError
	sessionID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	wrappedErr := ErrSessionNotFound

	sessionErr := &SessionError{
		SessionID: sessionID,
		Op:        "Get",
		Err:       wrappedErr,
	}

	// Test errors.Is
	if !errors.Is(sessionErr, wrappedErr) {
		t.Error("errors.Is should return true for wrapped error")
	}

	// Test that we can still access the SessionError
	var target *SessionError
	if !errors.As(sessionErr, &target) {
		t.Error("errors.As should return true for SessionError")
	}
	if target.SessionID != sessionID {
		t.Errorf("SessionID not preserved: got %v, want %v", target.SessionID, sessionID)
	}
	if target.Op != "Get" {
		t.Errorf("Op not preserved: got %q, want %q", target.Op, "Get")
	}
}
