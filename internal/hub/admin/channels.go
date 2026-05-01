package admin

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// ListChannels returns a paginated channel list. opts.Extra honors Hub's
// `status` (-1 all / 1 enabled / 0 disabled), `type`, `tag_mode`, `id_sort`.
func (c *Client) ListChannels(ctx context.Context, opts *ListOpts) (*ChannelPage, error) {
	var out ChannelPage
	if err := c.do(ctx, http.MethodGet, "/api/channel/", opts.pageQuery(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SearchChannels is the Hub search endpoint (matches name + tag + remark).
func (c *Client) SearchChannels(ctx context.Context, keyword string) ([]Channel, error) {
	q := newQuery("keyword", keyword)
	var out []Channel
	if err := c.do(ctx, http.MethodGet, "/api/channel/search", q, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetChannel returns a single channel by ID.
func (c *Client) GetChannel(ctx context.Context, id int) (*Channel, error) {
	var out Channel
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/channel/%d", id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AddChannel creates a new channel. Hub wraps the payload in
// `AddChannelRequest{Channel, Mode, MultiKeyMode}` — for the simple case we
// pass a flat map and let Hub apply defaults.
func (c *Client) AddChannel(ctx context.Context, input CreateChannelInput) error {
	body := map[string]any{"channel": input}
	return c.do(ctx, http.MethodPost, "/api/channel/", nil, body, nil)
}

// UpdateChannel sends a PUT with the partial channel struct (Hub merges).
func (c *Client) UpdateChannel(ctx context.Context, channel map[string]any) error {
	return c.do(ctx, http.MethodPut, "/api/channel/", nil, channel, nil)
}

// DeleteChannel removes a single channel.
func (c *Client) DeleteChannel(ctx context.Context, id int) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/channel/%d", id), nil, nil, nil)
}

// DeleteChannelBatch removes multiple channels in one call.
func (c *Client) DeleteChannelBatch(ctx context.Context, ids []int) error {
	body := map[string]any{"ids": ids}
	return c.do(ctx, http.MethodPost, "/api/channel/batch", nil, body, nil)
}

// TestChannel runs a connectivity test against a single channel.
func (c *Client) TestChannel(ctx context.Context, id int, model string) error {
	q := newQuery("model", model)
	return c.do(ctx, http.MethodGet, fmt.Sprintf("/api/channel/test/%d", id), q, nil, nil)
}

// CopyChannel duplicates a channel (Hub appends a copy suffix to the name).
func (c *Client) CopyChannel(ctx context.Context, id int) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/api/channel/copy/%d", id), nil, nil, nil)
}

// newQuery returns a url.Values with one key set, or nil when v is empty.
// Used by endpoints that take a single query param so callers don't have to
// build url.Values inline.
func newQuery(k, v string) url.Values {
	if v == "" {
		return nil
	}
	q := url.Values{}
	q.Set(k, v)
	return q
}

// Channel status integer values exposed so callers don't import Hub's
// constant package directly. Hub also has auto-disabled (3) and multi-key
// partial states; surface those if a future page needs to render them.
const (
	ChannelStatusEnabled  = 1
	ChannelStatusDisabled = 2
)
