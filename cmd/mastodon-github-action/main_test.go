package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPostStatus(t *testing.T) {
	tests := []struct {
		name           string
		status         MastodonStatus
		mockResponse   string
		mockStatusCode int
		wantErr        bool
	}{
		{
			name: "successful post",
			status: MastodonStatus{
				Status:     "Test status",
				Visibility: string(VisibilityPublic),
			},
			mockResponse:   `{"id":"123","url":"http://example.com/status/123","content":"Test status","created_at":"2020-01-01T00:00:00Z","visibility":"public"}`,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "error from server",
			status: MastodonStatus{
				Status:     "Test status",
				Visibility: string(VisibilityPublic),
			},
			mockResponse:   `{"error":"Internal server error"}`,
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
		{
			name: "successful unlisted post",
			status: MastodonStatus{
				Status:     "Unlisted status",
				Visibility: string(VisibilityUnlisted),
			},
			mockResponse:   `{"id":"124","url":"http://example.com/status/124","content":"Unlisted status","created_at":"2020-01-02T00:00:00Z","visibility":"unlisted"}`,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "successful private post",
			status: MastodonStatus{
				Status:     "Private status",
				Visibility: string(VisibilityPrivate),
			},
			mockResponse:   `{"id":"125","url":"http://example.com/status/125","content":"Private status","created_at":"2020-01-03T00:00:00Z","visibility":"private"}`,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "successful direct post",
			status: MastodonStatus{
				Status:     "Direct message",
				Visibility: string(VisibilityDirect),
			},
			mockResponse:   `{"id":"126","url":"http://example.com/status/126","content":"Direct message","created_at":"2020-01-04T00:00:00Z","visibility":"direct"}`,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "successful post with sensitive content",
			status: MastodonStatus{
				Status:     "Sensitive status",
				Visibility: string(VisibilityPublic),
				Sensitive:  true,
			},
			mockResponse:   `{"id":"127","url":"http://example.com/status/127","content":"Sensitive status","created_at":"2020-01-05T00:00:00Z","visibility":"public"}`,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "successful post with spoiler text",
			status: MastodonStatus{
				Status:      "Spoiler status",
				Visibility:  string(VisibilityPublic),
				SpoilerText: "Spoiler alert!",
			},
			mockResponse:   `{"id":"128","url":"http://example.com/status/128","content":"Spoiler status","created_at":"2020-01-06T00:00:00Z","visibility":"public"}`,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "successful scheduled post",
			status: MastodonStatus{
				Status:      "Scheduled status",
				Visibility:  string(VisibilityPublic),
				ScheduledAt: "2020-01-07T00:00:00Z",
			},
			mockResponse:   `{"id":"129","scheduled_at":"2020-01-07T00:00:00Z"}`,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "post with invalid visibility",
			status: MastodonStatus{
				Status:     "Bad visibility status",
				Visibility: "invalid",
			},
			mockResponse:   `{"error":"Invalid visibility"}`,
			mockStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name: "API rate limit exceeded",
			status: MastodonStatus{
				Status:     "Rate limit status",
				Visibility: string(VisibilityPublic),
			},
			mockResponse:   `{"error":"Rate limit exceeded"}`,
			mockStatusCode: http.StatusTooManyRequests,
			wantErr:        true,
		},
		{
			name: "API unauthorized access",
			status: MastodonStatus{
				Status:     "Unauthorized status",
				Visibility: string(VisibilityPublic),
			},
			mockResponse:   `{"error":"Unauthorized"}`,
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name: "invalid scheduled_at format",
			status: MastodonStatus{
				Status:      "Invalid scheduled_at format",
				Visibility:  string(VisibilityPublic),
				ScheduledAt: "invalid-date",
			},
			mockResponse:   ``, // No mock response needed as the error should occur before the API call
			mockStatusCode: 0,  // Status code is irrelevant in this case
			wantErr:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.mockStatusCode)
				fmt.Fprintln(w, tc.mockResponse)
			}))
			defer mockServer.Close()

			args := ActionInputs{
				URL:         mockServer.URL, // Use mock server URL
				AccessToken: "fake-token",
				Message:     tc.status.Status,
				Visibility:  tc.status.Visibility,
				Sensitive:   tc.status.Sensitive,
				SpoilerText: tc.status.SpoilerText,
				Language:    tc.status.Language,
				ScheduledAt: tc.status.ScheduledAt,
			}

			err := postStatus(args.URL, args.AccessToken, tc.status)

			if (err != nil) != tc.wantErr {
				t.Errorf("postStatus() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestParseMediaPaths(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "empty string", input: "", want: nil},
		{name: "whitespace only", input: "   ", want: nil},
		{name: "single path", input: "/tmp/image.png", want: []string{"/tmp/image.png"}},
		{name: "multiple paths", input: "/tmp/a.png, /tmp/b.jpg, /tmp/c.gif", want: []string{"/tmp/a.png", "/tmp/b.jpg", "/tmp/c.gif"}},
		{name: "paths with extra whitespace", input: "  /tmp/a.png ,  /tmp/b.jpg  ", want: []string{"/tmp/a.png", "/tmp/b.jpg"}},
		{name: "trailing comma", input: "/tmp/a.png,", want: []string{"/tmp/a.png"}},
		{name: "empty entries between commas", input: "/tmp/a.png,,/tmp/b.png", want: []string{"/tmp/a.png", "/tmp/b.png"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseMediaPaths(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("parseMediaPaths(%q) = %v, want %v", tc.input, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("parseMediaPaths(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestParseMediaDescriptions(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "empty string", input: "", want: nil},
		{name: "whitespace only", input: "   ", want: nil},
		{name: "single description", input: "A nice photo", want: []string{"A nice photo"}},
		{name: "multiple descriptions", input: "Photo 1, Photo 2, Photo 3", want: []string{"Photo 1", "Photo 2", "Photo 3"}},
		{name: "descriptions with extra whitespace", input: "  Photo 1 ,  Photo 2  ", want: []string{"Photo 1", "Photo 2"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseMediaDescriptions(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("parseMediaDescriptions(%q) = %v, want %v", tc.input, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("parseMediaDescriptions(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestResolveDescription(t *testing.T) {
	tests := []struct {
		name         string
		index        int
		descriptions []string
		want         string
	}{
		{name: "nil descriptions", index: 0, descriptions: nil, want: ""},
		{name: "empty descriptions", index: 0, descriptions: []string{}, want: ""},
		{name: "exact match at index", index: 1, descriptions: []string{"first", "second", "third"}, want: "second"},
		{name: "single description applies to all", index: 2, descriptions: []string{"shared desc"}, want: "shared desc"},
		{name: "index out of range with multiple", index: 5, descriptions: []string{"a", "b"}, want: ""},
		{name: "empty string at index falls back to single", index: 1, descriptions: []string{"fallback", ""}, want: ""},
		{name: "single empty description", index: 0, descriptions: []string{""}, want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveDescription(tc.index, tc.descriptions)
			if got != tc.want {
				t.Errorf("resolveDescription(%d, %v) = %q, want %q", tc.index, tc.descriptions, got, tc.want)
			}
		})
	}
}

func TestUploadMedia(t *testing.T) {
	tests := []struct {
		name           string
		description    string
		mockStatusCode int
		mockResponse   string
		wantErr        bool
		wantID         string
	}{
		{
			name:           "successful sync upload (200)",
			description:    "test image",
			mockStatusCode: http.StatusOK,
			mockResponse:   `{"id":"media123","type":"image","url":"https://example.com/media/123.png"}`,
			wantErr:        false,
			wantID:         "media123",
		},
		{
			name:           "upload without description",
			description:    "",
			mockStatusCode: http.StatusOK,
			mockResponse:   `{"id":"media456","type":"image","url":"https://example.com/media/456.png"}`,
			wantErr:        false,
			wantID:         "media456",
		},
		{
			name:           "server error",
			description:    "test",
			mockStatusCode: http.StatusInternalServerError,
			mockResponse:   `{"error":"Internal server error"}`,
			wantErr:        true,
		},
		{
			name:           "unauthorized",
			description:    "test",
			mockStatusCode: http.StatusUnauthorized,
			mockResponse:   `{"error":"Unauthorized"}`,
			wantErr:        true,
		},
		{
			name:           "unprocessable entity",
			description:    "test",
			mockStatusCode: http.StatusUnprocessableEntity,
			mockResponse:   `{"error":"Validation failed: File content type is invalid"}`,
			wantErr:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "test-media-*.png")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpFile.Name())
			tmpFile.Write([]byte("fake image content"))
			tmpFile.Close()

			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if !strings.HasSuffix(r.URL.Path, "/api/v2/media") {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer fake-token" {
					t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
				}
				if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
					t.Errorf("expected multipart/form-data content type, got %s", r.Header.Get("Content-Type"))
				}

				err := r.ParseMultipartForm(10 << 20)
				if err != nil {
					t.Errorf("failed to parse multipart form: %v", err)
				}

				file, header, err := r.FormFile("file")
				if err != nil {
					t.Errorf("missing file field: %v", err)
				} else {
					defer file.Close()
					if header.Filename != filepath.Base(tmpFile.Name()) {
						t.Errorf("unexpected filename: %s", header.Filename)
					}
					content, _ := io.ReadAll(file)
					if string(content) != "fake image content" {
						t.Errorf("unexpected file content: %s", string(content))
					}
				}

				if tc.description != "" {
					desc := r.FormValue("description")
					if desc != tc.description {
						t.Errorf("expected description %q, got %q", tc.description, desc)
					}
				}

				w.WriteHeader(tc.mockStatusCode)
				fmt.Fprint(w, tc.mockResponse)
			}))
			defer mockServer.Close()

			id, err := uploadMedia(mockServer.URL, "fake-token", tmpFile.Name(), tc.description)
			if (err != nil) != tc.wantErr {
				t.Errorf("uploadMedia() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && id != tc.wantID {
				t.Errorf("uploadMedia() id = %q, want %q", id, tc.wantID)
			}
		})
	}
}

func TestUploadMediaAsyncProcessing(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-video-*.mp4")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("fake video content"))
	tmpFile.Close()

	pollCount := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/api/v2/media") {
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"id":"async-media-1","type":"video","url":null}`)
			return
		}

		if r.Method == "GET" && strings.Contains(r.URL.Path, "/api/v1/media/async-media-1") {
			pollCount++
			if pollCount < 3 {
				w.WriteHeader(http.StatusPartialContent)
				return
			}
			url := "https://example.com/media/video.mp4"
			resp := MediaAttachmentResponse{ID: "async-media-1", Type: "video", URL: &url}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	id, err := uploadMedia(mockServer.URL, "fake-token", tmpFile.Name(), "test video")
	if err != nil {
		t.Fatalf("uploadMedia() unexpected error: %v", err)
	}
	if id != "async-media-1" {
		t.Errorf("uploadMedia() id = %q, want %q", id, "async-media-1")
	}
	if pollCount < 3 {
		t.Errorf("expected at least 3 poll attempts, got %d", pollCount)
	}
}

func TestUploadMediaFileNotFound(t *testing.T) {
	_, err := uploadMedia("http://localhost", "token", "/nonexistent/file.png", "")
	if err == nil {
		t.Error("uploadMedia() expected error for nonexistent file, got nil")
	}
}

func TestPostStatusWithMediaIDs(t *testing.T) {
	var receivedBody map[string]interface{}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"id":"status-1","url":"https://example.com/@user/status-1","content":"Hello","created_at":"2020-01-01T00:00:00Z","visibility":"public"}`)
	}))
	defer mockServer.Close()

	status := MastodonStatus{
		Status:     "Hello with media",
		Visibility: string(VisibilityPublic),
		MediaIDs:   []string{"media-1", "media-2"},
	}

	err := postStatus(mockServer.URL, "fake-token", status)
	if err != nil {
		t.Fatalf("postStatus() unexpected error: %v", err)
	}

	mediaIDs, ok := receivedBody["media_ids"].([]interface{})
	if !ok {
		t.Fatal("expected media_ids in request body")
	}
	if len(mediaIDs) != 2 {
		t.Fatalf("expected 2 media_ids, got %d", len(mediaIDs))
	}
	if mediaIDs[0] != "media-1" || mediaIDs[1] != "media-2" {
		t.Errorf("unexpected media_ids: %v", mediaIDs)
	}
}

func TestPostStatusWithoutMessageButWithMedia(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"id":"status-2","url":"https://example.com/@user/status-2","content":"","created_at":"2020-01-01T00:00:00Z","visibility":"public"}`)
	}))
	defer mockServer.Close()

	status := MastodonStatus{
		Status:     "",
		Visibility: string(VisibilityPublic),
		MediaIDs:   []string{"media-1"},
	}

	err := postStatus(mockServer.URL, "fake-token", status)
	if err != nil {
		t.Fatalf("postStatus() with empty status but media should succeed, got: %v", err)
	}
}
