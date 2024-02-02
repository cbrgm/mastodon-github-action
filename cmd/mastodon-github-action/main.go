package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
)

// Application metadata variables store runtime and build information
// for diagnostic purposes.
var (
	Version   string              // Version denotes the current version of the application.
	Revision  string              // Revision indicates the git commit hash the binary was built from.
	GoVersion = runtime.Version() // GoVersion records the version of Go runtime used to build the application.
	StartTime = time.Now()        // StartTime captures the time when the application was started.
)

// VisibilityType represents the Mastodon status visibility levels as type string.
type VisibilityType string

// Constants for Mastodon status visibility options.
const (
	VisibilityPublic   VisibilityType = "public"
	VisibilityUnlisted VisibilityType = "unlisted"
	VisibilityPrivate  VisibilityType = "private"
	VisibilityDirect   VisibilityType = "direct"
)

// IsValid checks if the visibility type is a valid Mastodon visibility option.
func (vt VisibilityType) IsValid() bool {
	return vt == VisibilityPublic || vt == VisibilityUnlisted || vt == VisibilityPrivate || vt == VisibilityDirect
}

// MastodonStatus defines the structure for status messages to be sent to Mastodon.
type MastodonStatus struct {
	Status      string `json:"status"`
	Visibility  string `json:"visibility"`
	Sensitive   bool   `json:"sensitive,omitempty"`
	SpoilerText string `json:"spoiler_text,omitempty"`
	Language    string `json:"language,omitempty"`
	ScheduledAt string `json:"scheduled_at,omitempty"`
}

// ActionInputs collects all user inputs required for posting a status.
type ActionInputs struct {
	URL         string `arg:"--url,required" env:"MASTODON_URL"`                   // Mastodon instance URL.
	AccessToken string `arg:"--access-token,required" env:"MASTODON_ACCESS_TOKEN"` // User access token for authentication.
	Message     string `arg:"--message,required" env:"MASTODON_MESSAGE"`           // The status message content.
	Visibility  string `arg:"--visibility" env:"MASTODON_VISIBILITY"`              // Visibility of the status.
	Sensitive   bool   `arg:"--sensitive" env:"MASTODON_SENSITIVE"`                // Flag to mark status as sensitive.
	SpoilerText string `arg:"--spoiler-text" env:"MASTODON_SPOILER_TEXT"`          // Additional content warning text.
	Language    string `arg:"--language" env:"MASTODON_LANGUAGE"`                  // Language of the status.
	ScheduledAt string `arg:"--scheduled-at" env:"MASTODON_SCHEDULED_AT"`          // Time to schedule the status.
}

// StatusResponse models the response returned by Mastodon after posting a status.
type StatusResponse struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	Content    string `json:"content"`
	CreatedAt  string `json:"created_at"`
	Visibility string `json:"visibility"`
}

// ScheduledStatusResponse captures the response for a successfully scheduled status post.
type ScheduledStatusResponse struct {
	ID          string    `json:"id"`
	ScheduledAt time.Time `json:"scheduled_at"`
}

func main() {
	var args ActionInputs
	arg.MustParse(&args) // Parses and validates command-line arguments.

	// Ensures the status message is not empty.
	if strings.TrimSpace(args.Message) == "" {
		log.Fatal("Status message cannot be empty")
	}

	// Sets default visibility to "public" if not specified.
	if args.Visibility == "" {
		args.Visibility = string(VisibilityPublic)
	}

	// Validates the provided visibility against Mastodon's accepted values.
	if !VisibilityType(args.Visibility).IsValid() {
		log.Fatalf("Invalid visibility: %s", args.Visibility)
	}

	// Parse and validate the scheduled time, if provided.
	scheduledAt, err := parseScheduledAt(args.ScheduledAt)
	if err != nil {
		log.Fatalf("Scheduled at error: %v", err)
	}

	// Constructs the status payload and sends it to Mastodon.
	status := MastodonStatus{
		Status:      args.Message,
		Visibility:  args.Visibility,
		Sensitive:   args.Sensitive,
		SpoilerText: args.SpoilerText,
		Language:    args.Language,
		ScheduledAt: scheduledAt,
	}

	if err := postStatus(args.URL, args.AccessToken, status); err != nil {
		log.Fatalf("Error posting status: %v", err)
	}
}

// postStatus sends a status update to the specified Mastodon instance using the provided access token.
func postStatus(url, accessToken string, status MastodonStatus) error {
	payload, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("marshaling error: %w", err)
	}

	apiURL := fmt.Sprintf("%s/api/v1/statuses", url)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("request creation error: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-Type", "application/json")

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("API call error: %w", err)
	}

	//nolint: errcheck
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) // Reading body for debug
		return fmt.Errorf("API response: %s, Body: %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body error: %w", err)
	}

	// Determine response type (immediate or scheduled) and output accordingly
	if !strings.Contains(status.ScheduledAt, "T") {
		var statusResponse StatusResponse
		if err := json.Unmarshal(body, &statusResponse); err != nil {
			return fmt.Errorf("unmarshaling status response error: %w", err)
		}
		setActionOutputs(map[string]string{
			"id":  statusResponse.ID,
			"url": statusResponse.URL,
		})
		log.Printf("Status Posted: %s, URL: %s", statusResponse.ID, statusResponse.URL)
	} else {
		var scheduledResponse ScheduledStatusResponse
		if err := json.Unmarshal(body, &scheduledResponse); err != nil {
			return fmt.Errorf("unmarshaling scheduled response error: %w", err)
		}
		setActionOutputs(map[string]string{
			"id":           scheduledResponse.ID,
			"scheduled_at": scheduledResponse.ScheduledAt.String(),
		})
		log.Printf("Scheduled Status ID: %s, Scheduled At: %s", scheduledResponse.ID, scheduledResponse.ScheduledAt)
	}

	return nil
}

// parseScheduledAt converts a user-friendly date/time input into an ISO 8601 formatted string,
// validating that it is at least 5 minutes in the future.
func parseScheduledAt(input string) (string, error) {
	if input == "" {
		return "", nil // No scheduling requested
	}

	t, err := time.Parse("2006-01-02 15:04", input)
	if err != nil {
		return "", fmt.Errorf("invalid date format: %w", err)
	}

	if time.Until(t) < 5*time.Minute {
		return "", fmt.Errorf("scheduled time must be at least 5 minutes in the future")
	}

	return t.Format(time.RFC3339), nil
}
