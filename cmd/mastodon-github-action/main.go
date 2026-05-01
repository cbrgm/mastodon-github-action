package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
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
	Status      string   `json:"status"`
	Visibility  string   `json:"visibility"`
	Sensitive   bool     `json:"sensitive,omitempty"`
	SpoilerText string   `json:"spoiler_text,omitempty"`
	Language    string   `json:"language,omitempty"`
	ScheduledAt string   `json:"scheduled_at,omitempty"`
	MediaIDs    []string `json:"media_ids,omitempty"`
}

// ActionInputs collects all user inputs required for posting a status.
type ActionInputs struct {
	URL               string `arg:"--url,required, env:MASTODON_URL"`                      // Mastodon instance URL.
	AccessToken       string `arg:"--access-token,required, env:MASTODON_ACCESS_TOKEN"`    // User access token for authentication.
	Message           string `arg:"--message, env:MASTODON_MESSAGE"`                       // The status message content.
	Visibility        string `arg:"--visibility, env:MASTODON_VISIBILITY"`                 // Visibility of the status.
	Sensitive         bool   `arg:"--sensitive, env:MASTODON_SENSITIVE"`                   // Flag to mark status as sensitive.
	SpoilerText       string `arg:"--spoiler-text, env:MASTODON_SPOILER_TEXT"`             // Additional content warning text.
	Language          string `arg:"--language, env:MASTODON_LANGUAGE"`                     // Language of the status.
	ScheduledAt       string `arg:"--scheduled-at, env:MASTODON_SCHEDULED_AT"`             // Time to schedule the status.
	MediaPaths        string `arg:"--media-paths, env:MASTODON_MEDIA_PATHS"`               // Comma-separated media file paths.
	MediaDescriptions string `arg:"--media-descriptions, env:MASTODON_MEDIA_DESCRIPTIONS"` // Comma-separated alt text descriptions.
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

const maxMediaAttachments = 4

func main() {
	var args ActionInputs
	arg.MustParse(&args)

	mediaPaths := parseMediaPaths(args.MediaPaths)
	hasMedia := len(mediaPaths) > 0

	if strings.TrimSpace(args.Message) == "" && !hasMedia {
		log.Fatal("Status message cannot be empty when no media is attached")
	}

	if len(mediaPaths) > maxMediaAttachments {
		log.Fatalf("Too many media attachments: %d (maximum is %d)", len(mediaPaths), maxMediaAttachments)
	}

	if args.Visibility == "" {
		args.Visibility = string(VisibilityPublic)
	}

	if !VisibilityType(args.Visibility).IsValid() {
		log.Fatalf("Invalid visibility: %s", args.Visibility)
	}

	scheduledAt, err := parseScheduledAt(args.ScheduledAt)
	if err != nil {
		log.Fatalf("Scheduled at error: %v", err)
	}

	var mediaIDs []string
	if hasMedia {
		descriptions := parseMediaDescriptions(args.MediaDescriptions)
		for i, path := range mediaPaths {
			desc := resolveDescription(i, descriptions)
			id, err := uploadMedia(args.URL, args.AccessToken, path, desc)
			if err != nil {
				log.Fatalf("Error uploading media %q: %v", path, err)
			}
			mediaIDs = append(mediaIDs, id)
			log.Printf("Uploaded media %q: ID %s", path, id)
		}
	}

	status := MastodonStatus{
		Status:      args.Message,
		Visibility:  args.Visibility,
		Sensitive:   args.Sensitive,
		SpoilerText: args.SpoilerText,
		Language:    args.Language,
		ScheduledAt: scheduledAt,
		MediaIDs:    mediaIDs,
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
		return "", nil
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

// parseMediaPaths splits a comma-separated list of file paths into a slice,
// trimming whitespace and discarding empty entries.
func parseMediaPaths(input string) []string {
	if strings.TrimSpace(input) == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	var paths []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			paths = append(paths, trimmed)
		}
	}
	return paths
}

// parseMediaDescriptions splits a comma-separated list of descriptions.
func parseMediaDescriptions(input string) []string {
	if strings.TrimSpace(input) == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	var descs []string
	for _, p := range parts {
		descs = append(descs, strings.TrimSpace(p))
	}
	return descs
}

// resolveDescription returns the alt text for a media item at the given index.
// If a specific description exists for that index, it is used. If only one
// description is provided, it is used for all items. Otherwise returns empty string.
func resolveDescription(index int, descriptions []string) string {
	if index < len(descriptions) && descriptions[index] != "" {
		return descriptions[index]
	}
	if len(descriptions) == 1 && descriptions[0] != "" {
		return descriptions[0]
	}
	return ""
}

// MediaAttachmentResponse represents the response from the Mastodon media upload API.
type MediaAttachmentResponse struct {
	ID   string  `json:"id"`
	Type string  `json:"type"`
	URL  *string `json:"url"`
}

// uploadMedia uploads a media file to the Mastodon instance and returns the attachment ID.
// It uses the v2 media endpoint which may return 202 for async processing.
func uploadMedia(instanceURL, accessToken, filePath, description string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("opening file: %w", err)
	}
	//nolint: errcheck
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", fmt.Errorf("creating form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("copying file content: %w", err)
	}

	if description != "" {
		if err := writer.WriteField("description", description); err != nil {
			return "", fmt.Errorf("writing description field: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("closing multipart writer: %w", err)
	}

	apiURL := fmt.Sprintf("%s/api/v2/media", instanceURL)
	req, err := http.NewRequest("POST", apiURL, body)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API call: %w", err)
	}
	//nolint: errcheck
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("API response: %s, Body: %s", resp.Status, string(respBody))
	}

	var attachment MediaAttachmentResponse
	if err := json.Unmarshal(respBody, &attachment); err != nil {
		return "", fmt.Errorf("unmarshaling response: %w", err)
	}

	// 202 means async processing — poll until ready
	if resp.StatusCode == http.StatusAccepted {
		if err := pollMediaProcessing(instanceURL, accessToken, attachment.ID); err != nil {
			return "", err
		}
	}

	return attachment.ID, nil
}

// pollMediaProcessing polls GET /api/v1/media/:id until the media is processed.
func pollMediaProcessing(instanceURL, accessToken, mediaID string) error {
	apiURL := fmt.Sprintf("%s/api/v1/media/%s", instanceURL, mediaID)
	client := http.Client{Timeout: 10 * time.Second}

	for attempts := 0; attempts < 30; attempts++ {
		time.Sleep(2 * time.Second)

		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return fmt.Errorf("creating poll request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("polling media status: %w", err)
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close() //nolint: errcheck

		if resp.StatusCode == http.StatusOK {
			var attachment MediaAttachmentResponse
			if err := json.Unmarshal(body, &attachment); err != nil {
				return fmt.Errorf("unmarshaling poll response: %w", err)
			}
			if attachment.URL != nil {
				return nil
			}
		} else if resp.StatusCode != http.StatusPartialContent {
			return fmt.Errorf("unexpected poll response: %s, Body: %s", resp.Status, string(body))
		}
	}
	return fmt.Errorf("media processing timed out for ID %s", mediaID)
}
