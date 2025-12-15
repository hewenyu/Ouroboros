package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestValidateSignature(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"test": "payload"}`)

	// Generate valid signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	validSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name        string
		body        []byte
		signature   string
		secret      string
		wantErr     bool
		expectedErr error
	}{
		{
			name:      "valid signature",
			body:      body,
			signature: validSig,
			secret:    secret,
			wantErr:   false,
		},
		{
			name:        "missing signature header",
			body:        body,
			signature:   "",
			secret:      secret,
			wantErr:     true,
			expectedErr: ErrMissingSignature,
		},
		{
			name:        "invalid signature format - no prefix",
			body:        body,
			signature:   "invalid-signature",
			secret:      secret,
			wantErr:     true,
			expectedErr: ErrInvalidSignatureFormat,
		},
		{
			name:        "invalid hex signature",
			body:        body,
			signature:   "sha256=not-hex",
			secret:      secret,
			wantErr:     true,
			expectedErr: ErrInvalidHexSignature,
		},
		{
			name:        "signature mismatch - wrong secret",
			body:        body,
			signature:   validSig,
			secret:      "wrong-secret",
			wantErr:     true,
			expectedErr: ErrSignatureMismatch,
		},
		{
			name:        "signature mismatch - modified body",
			body:        []byte(`{"test": "modified"}`),
			signature:   validSig,
			secret:      secret,
			wantErr:     true,
			expectedErr: ErrSignatureMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSignature(tt.body, tt.signature, tt.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSignature() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.expectedErr != nil && err != tt.expectedErr {
				t.Errorf("ValidateSignature() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestWorkflowRunPayloadParsing(t *testing.T) {
	// Test that the payload struct can be unmarshaled correctly
	payload := WorkflowRunPayload{
		Action: "completed",
		WorkflowRun: WorkflowRun{
			ID:         12345,
			HeadSHA:    "abc123",
			HeadBranch: "main",
			Conclusion: "success",
			Status:     "completed",
			Name:       "CI",
		},
		Repository: Repository{
			FullName: "owner/repo",
			Name:     "repo",
		},
	}

	if payload.Action != "completed" {
		t.Errorf("Expected action 'completed', got '%s'", payload.Action)
	}

	if payload.WorkflowRun.ID != 12345 {
		t.Errorf("Expected run ID 12345, got %d", payload.WorkflowRun.ID)
	}

	if payload.WorkflowRun.HeadSHA != "abc123" {
		t.Errorf("Expected head SHA 'abc123', got '%s'", payload.WorkflowRun.HeadSHA)
	}

	if payload.Repository.FullName != "owner/repo" {
		t.Errorf("Expected repo 'owner/repo', got '%s'", payload.Repository.FullName)
	}
}
