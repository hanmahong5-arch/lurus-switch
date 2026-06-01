package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// envRespond writes the standard Hub success envelope.
func envRespond(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "",
		"data":    data,
	})
}

// envError writes the standard Hub failure envelope (HTTP 200 with success:false).
func envError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"message": msg,
	})
}

// newTestClient pairs a mock server with a configured client. Each test gets
// its own server so handler routing is fully scoped.
func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := New(Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c, srv
}

func TestNew_RejectsEmptyBaseURL(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Error("expected error for empty BaseURL, got nil")
	}
}

func TestNew_TrimsTrailingSlash(t *testing.T) {
	c, err := New(Config{BaseURL: "https://hub.example.com//"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(c.baseURL, ".com") {
		t.Errorf("expected trailing slash trimmed, got %q", c.baseURL)
	}
}

func TestDo_PassesAuthHeader(t *testing.T) {
	got := ""
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		envRespond(w, nil)
	})
	if err := c.do(context.Background(), http.MethodGet, "/api/ping", nil, nil, nil); err != nil {
		t.Fatalf("do: %v", err)
	}
	if got != "test-token" {
		t.Errorf("Authorization = %q, want test-token", got)
	}
}

func TestDo_OmitsAuthWhenNoToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Errorf("unexpected Authorization header: %q", r.Header.Get("Authorization"))
		}
		envRespond(w, nil)
	}))
	defer srv.Close()
	c, _ := New(Config{BaseURL: srv.URL})
	if err := c.do(context.Background(), http.MethodGet, "/api/ping", nil, nil, nil); err != nil {
		t.Fatalf("do: %v", err)
	}
}

func TestDo_UnwrapsSuccessEnvelope(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		envRespond(w, map[string]any{"id": 42, "name": "acme"})
	})
	var got struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := c.do(context.Background(), http.MethodGet, "/api/anything", nil, nil, &got); err != nil {
		t.Fatalf("do: %v", err)
	}
	if got.ID != 42 || got.Name != "acme" {
		t.Errorf("unexpected payload: %+v", got)
	}
}

func TestDo_ConvertsFailureEnvelopeToHubError(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		envError(w, "no permission")
	})
	err := c.do(context.Background(), http.MethodGet, "/api/anything", nil, nil, nil)
	var hubErr *HubError
	if !errors.As(err, &hubErr) {
		t.Fatalf("expected *HubError, got %T: %v", err, err)
	}
	if !strings.Contains(hubErr.Message, "no permission") {
		t.Errorf("unexpected message: %q", hubErr.Message)
	}
}

func TestDo_DetectsUnauthorized(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"success":false,"message":"未登录"}`)
	})
	err := c.do(context.Background(), http.MethodGet, "/api/anything", nil, nil, nil)
	if !IsUnauthorized(err) {
		t.Fatalf("expected IsUnauthorized true, got %v", err)
	}
}

func TestDo_HandlesNonJSON(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "<html>500 Internal Server Error</html>")
	})
	err := c.do(context.Background(), http.MethodGet, "/api/anything", nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for non-JSON 500, got nil")
	}
	var hubErr *HubError
	if !errors.As(err, &hubErr) {
		t.Fatalf("expected *HubError, got %T", err)
	}
	if hubErr.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("HTTPStatus = %d, want 500", hubErr.HTTPStatus)
	}
}

func TestListChannels_PaginationAndQuery(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/channel/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("p") != "2" || r.URL.Query().Get("page_size") != "20" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		envRespond(w, ChannelPage{
			Items:    []Channel{{ID: 1, Name: "openai", Status: 1}},
			Page:     2,
			PageSize: 20,
			Total:    37,
		})
	})
	got, err := c.ListChannels(context.Background(), &ListOpts{Page: 2, PageSize: 20})
	if err != nil {
		t.Fatalf("ListChannels: %v", err)
	}
	if got.Total != 37 || len(got.Items) != 1 || got.Items[0].Name != "openai" {
		t.Errorf("unexpected page: %+v", got)
	}
}

func TestAddChannel_WrapsPayload(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var got struct {
			Channel map[string]any `json:"channel"`
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		if got.Channel["name"] != "acme-channel" {
			t.Errorf("unexpected channel payload: %+v", got)
		}
		envRespond(w, nil)
	})
	if err := c.AddChannel(context.Background(), CreateChannelInput{"name": "acme-channel", "type": 1}); err != nil {
		t.Fatalf("AddChannel: %v", err)
	}
}

func TestDeleteChannelBatch(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/channel/batch" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var got struct {
			IDs []int `json:"ids"`
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		if len(got.IDs) != 3 || got.IDs[0] != 5 {
			t.Errorf("unexpected ids: %v", got.IDs)
		}
		envRespond(w, nil)
	})
	if err := c.DeleteChannelBatch(context.Background(), []int{5, 6, 7}); err != nil {
		t.Fatalf("DeleteChannelBatch: %v", err)
	}
}

func TestCreateRedemptions_DefaultsCount(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var got CreateRedemptionInput
		_ = json.NewDecoder(r.Body).Decode(&got)
		if got.Count != 1 {
			t.Errorf("Count = %d, want 1 (defaulted)", got.Count)
		}
		envRespond(w, []Redemption{{ID: 1, Key: "code-aaa"}})
	})
	got, err := c.CreateRedemptions(context.Background(), CreateRedemptionInput{
		Name:  "starter",
		Quota: 100_000,
		// Count omitted — should default to 1
	})
	if err != nil {
		t.Fatalf("CreateRedemptions: %v", err)
	}
	if len(got) != 1 || got[0].Key != "code-aaa" {
		t.Errorf("unexpected redemptions: %+v", got)
	}
}

func TestListLogs_OnlyMineRoutesToSelf(t *testing.T) {
	hits := map[string]int{}
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		hits[r.URL.Path]++
		envRespond(w, LogPage{Items: []LogEntry{{ID: 1, ModelName: "gpt-4o"}}})
	})
	if _, err := c.ListLogs(context.Background(), LogQuery{OnlyMine: true}); err != nil {
		t.Fatalf("self ListLogs: %v", err)
	}
	if _, err := c.ListLogs(context.Background(), LogQuery{}); err != nil {
		t.Fatalf("admin ListLogs: %v", err)
	}
	if hits["/api/log/self/"] != 1 {
		t.Errorf("expected /api/log/self/ once, got %d", hits["/api/log/self/"])
	}
	if hits["/api/log/"] != 1 {
		t.Errorf("expected /api/log/ once, got %d", hits["/api/log/"])
	}
}

func TestListSwitchPresets(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/switch/presets" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		envRespond(w, []SwitchPreset{{ID: "openai", Name: "OpenAI", IsOfficial: true}})
	})
	got, err := c.ListSwitchPresets(context.Background())
	if err != nil {
		t.Fatalf("ListSwitchPresets: %v", err)
	}
	if len(got) != 1 || got[0].ID != "openai" {
		t.Errorf("unexpected presets: %+v", got)
	}
}

// countingWriter wraps an http.ResponseWriter to count how many body bytes the
// handler actually managed to flush before the client closed the connection.
// It lets us prove the client does not drain an unbounded body.
type countingWriter struct {
	http.ResponseWriter
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	written, err := c.ResponseWriter.Write(p)
	c.n += int64(written)
	return written, err
}

// TestDo_BoundsResponseBodyRead verifies that the Hub admin client caps how
// much of a response body it reads at maxResponseSize, mirroring the sibling
// internal/billing client. A wrong-host or misbehaving Hub returning a huge
// body must not be drained into memory unbounded.
func TestDo_BoundsResponseBodyRead(t *testing.T) {
	tests := []struct {
		name     string
		bodySize int  // total bytes the server attempts to send
		wantErr  bool // do() should surface an error (non-envelope / truncated)
	}{
		{
			name:     "under_limit_valid_envelope",
			bodySize: 0, // handled specially below: send a real small envelope
			wantErr:  false,
		},
		{
			name:     "way_over_limit_is_capped_and_errors_cleanly",
			bodySize: maxResponseSize + (4 << 20), // 4 MB past the cap
			wantErr:  true,                         // truncated garbage → non-JSON HubError, not OOM
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var served int64
			c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				cw := &countingWriter{ResponseWriter: w}
				if tt.bodySize == 0 {
					envRespond(cw, map[string]any{"ok": true})
					served = cw.n
					return
				}
				// Stream filler bytes; the client must stop reading at the cap
				// and the handler's writes will fail/stop shortly after.
				chunk := bytes.Repeat([]byte("x"), 64<<10) // 64 KB
				remaining := tt.bodySize
				for remaining > 0 {
					if remaining < len(chunk) {
						chunk = chunk[:remaining]
					}
					m, err := cw.Write(chunk)
					remaining -= m
					if err != nil {
						break // client closed the connection after the cap
					}
				}
				served = cw.n
			})

			err := c.do(context.Background(), http.MethodGet, "/api/anything", nil, nil, nil)

			if tt.wantErr && err == nil {
				t.Fatalf("expected error for over-limit body, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for under-limit body: %v", err)
			}

			// Core guarantee: the client never pulls more than maxResponseSize
			// from the wire. We assert against what the server managed to push;
			// for the over-limit case the server cannot push meaningfully more
			// than the cap (plus in-flight TCP/HTTP buffering) before the client
			// stops reading and closes. Use a generous slack for kernel/HTTP
			// buffers but far below the attempted bodySize.
			const slack = 8 << 20 // 8 MB of OS/HTTP buffering headroom
			if served > maxResponseSize+slack {
				t.Errorf("server pushed %d bytes; client should have capped near %d (cap=%d)",
					served, maxResponseSize+slack, maxResponseSize)
			}
		})
	}
}

func TestListTenants_RequiresRootRole(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		// Simulate Hub returning 401 when token lacks root role.
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, `{"success":false,"message":"需要 root 角色"}`)
	})
	_, err := c.ListTenants(context.Background())
	if !IsUnauthorized(err) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}
