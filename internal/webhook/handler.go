// Package webhook provides GitHub webhook handling with HMAC-SHA256 validation.
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/hewenyu/Ouroboros/internal/database"
)

// truncate safely truncates a string to the specified length.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

var (
	// ErrInvalidSignatureFormat is returned when the signature header has an invalid format.
	ErrInvalidSignatureFormat = errors.New("invalid signature format")
	// ErrInvalidHexSignature is returned when the signature cannot be decoded from hex.
	ErrInvalidHexSignature = errors.New("invalid hex signature")
	// ErrSignatureMismatch is returned when the signature does not match.
	ErrSignatureMismatch = errors.New("signature mismatch")
	// ErrMissingSignature is returned when the signature header is missing.
	ErrMissingSignature = errors.New("missing signature header")
)

// WorkflowRunPayload represents the relevant parts of a GitHub workflow_run webhook payload.
type WorkflowRunPayload struct {
	Action      string      `json:"action"`
	WorkflowRun WorkflowRun `json:"workflow_run"`
	Repository  Repository  `json:"repository"`
}

// WorkflowRun represents the workflow run details.
type WorkflowRun struct {
	ID         int64  `json:"id"`
	HeadSHA    string `json:"head_sha"`
	HeadBranch string `json:"head_branch"`
	Conclusion string `json:"conclusion"`
	Status     string `json:"status"`
	Name       string `json:"name"`
	HTMLURL    string `json:"html_url"`
}

// Repository represents the repository information.
type Repository struct {
	FullName string `json:"full_name"`
	Name     string `json:"name"`
	Owner    Owner  `json:"owner"`
}

// Owner represents the repository owner.
type Owner struct {
	Login string `json:"login"`
}

// ValidateSignature validates the GitHub webhook signature using HMAC-SHA256.
// Uses constant-time comparison to prevent timing attacks.
func ValidateSignature(body []byte, signatureHeader string, secret string) error {
	if signatureHeader == "" {
		return ErrMissingSignature
	}

	if !strings.HasPrefix(signatureHeader, "sha256=") {
		return ErrInvalidSignatureFormat
	}

	signatureHex := strings.TrimPrefix(signatureHeader, "sha256=")
	signatureBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		return ErrInvalidHexSignature
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedMAC := mac.Sum(nil)

	// Use constant-time comparison to prevent timing attacks
	if !hmac.Equal(signatureBytes, expectedMAC) {
		return ErrSignatureMismatch
	}

	return nil
}

// EventProcessor processes webhook events asynchronously.
type EventProcessor struct {
	events chan WorkflowRunPayload
	store  *database.Store
	docker DockerVerifier
}

// DockerVerifier interface for container verification.
type DockerVerifier interface {
	VerifyDeployment(commitSHA string, serviceName string) (bool, string, error)
}

// NewEventProcessor creates a new event processor with a buffered channel.
func NewEventProcessor(store *database.Store, docker DockerVerifier, bufferSize int) *EventProcessor {
	ep := &EventProcessor{
		events: make(chan WorkflowRunPayload, bufferSize),
		store:  store,
		docker: docker,
	}
	return ep
}

// Start starts the worker goroutines for processing events.
func (ep *EventProcessor) Start(workers int) {
	for i := 0; i < workers; i++ {
		go ep.worker(i)
	}
	log.Printf("Started %d webhook event workers", workers)
}

// worker is the event processing goroutine.
func (ep *EventProcessor) worker(id int) {
	for event := range ep.events {
		ep.processEvent(id, event)
	}
}

// processEvent processes a single workflow run event.
func (ep *EventProcessor) processEvent(workerID int, event WorkflowRunPayload) {
	log.Printf("[Worker %d] Processing event: %s for run %d", workerID, event.Action, event.WorkflowRun.ID)

	// Only process completed successful builds
	if event.Action != "completed" || event.WorkflowRun.Conclusion != "success" {
		log.Printf("[Worker %d] Skipping event: action=%s, conclusion=%s",
			workerID, event.Action, event.WorkflowRun.Conclusion)
		return
	}

	// Create deployment log
	deploymentLog := &database.DeploymentLog{
		ID:        uuid.New().String(),
		TraceID:   event.WorkflowRun.ID,
		CommitSHA: event.WorkflowRun.HeadSHA,
		RepoName:  event.Repository.FullName,
		Branch:    event.WorkflowRun.HeadBranch,
		Status:    database.DeploymentStatusPending,
	}

	if err := ep.store.CreateDeploymentLog(deploymentLog); err != nil {
		log.Printf("[Worker %d] Failed to create deployment log: %v", workerID, err)
		return
	}

	// Verify deployment if docker verifier is available
	if ep.docker != nil {
		verified, logMsg, err := ep.docker.VerifyDeployment(event.WorkflowRun.HeadSHA, event.Repository.Name)
		if err != nil {
			deploymentLog.Status = database.DeploymentStatusFailed
			deploymentLog.Logs = fmt.Sprintf("Verification failed: %v", err)
		} else if verified {
			deploymentLog.Status = database.DeploymentStatusVerified
			deploymentLog.Logs = logMsg
		} else {
			deploymentLog.Status = database.DeploymentStatusFailed
			deploymentLog.Logs = logMsg
		}

		if err := ep.store.UpdateDeploymentLog(deploymentLog); err != nil {
			log.Printf("[Worker %d] Failed to update deployment log: %v", workerID, err)
		}
	}

	log.Printf("[Worker %d] Processed deployment for commit %s: %s",
		workerID, truncate(event.WorkflowRun.HeadSHA, 8), deploymentLog.Status)
}

// Enqueue adds an event to the processing queue.
func (ep *EventProcessor) Enqueue(event WorkflowRunPayload) {
	select {
	case ep.events <- event:
		log.Printf("Enqueued event for run %d", event.WorkflowRun.ID)
	default:
		log.Printf("Warning: Event queue full, dropping event for run %d", event.WorkflowRun.ID)
	}
}

// Close closes the event processor.
func (ep *EventProcessor) Close() {
	close(ep.events)
}

// Handler handles GitHub webhook HTTP requests.
type Handler struct {
	secret    string
	processor *EventProcessor
}

// NewHandler creates a new webhook handler.
func NewHandler(secret string, processor *EventProcessor) *Handler {
	return &Handler{
		secret:    secret,
		processor: processor,
	}
}

// ServeHTTP handles the webhook HTTP request.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Validate signature if secret is configured
	if h.secret != "" {
		signature := r.Header.Get("X-Hub-Signature-256")
		if err := ValidateSignature(body, signature, h.secret); err != nil {
			log.Printf("Signature validation failed: %v", err)
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// Check event type
	eventType := r.Header.Get("X-GitHub-Event")
	if eventType != "workflow_run" {
		// Accept but ignore non-workflow_run events
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, "Event type '%s' ignored", eventType)
		return
	}

	// Parse the payload
	var payload WorkflowRunPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("Failed to parse webhook payload: %v", err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Enqueue for async processing
	h.processor.Enqueue(payload)

	// Return 202 Accepted immediately
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, "Webhook received for run %d", payload.WorkflowRun.ID)
}
