// Package stream ports the basics of schwab-py's StreamClient
// (schwab/streaming.py): it fetches streaming parameters from the client's
// user preferences, performs the ADMIN LOGIN handshake over a websocket,
// issues subscriptions, and delivers incoming messages.
//
// Unlike the Python library — which dispatches messages to per-service
// handler callbacks that relabel numeric field keys — this port delivers
// each raw message to a single handler as it arrives. The services, commands,
// request envelope, and field-id ranges match schwab-py exactly.
package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/rayli09/schwab-cli/internal/client"
)

// Feed describes one subscribable streaming service.
type Feed struct {
	Service string
	// MaxField is the highest field id of the service's field enum in
	// schwab-py; field id ranges there are contiguous from 0.
	MaxField int
	// HasFields reports whether subscriptions for this service carry a
	// fields parameter.
	HasFields bool
}

// Feeds mirrors the subscription surfaces of schwab-py's StreamClient.
// Command names are the kebab-case forms used by the CLI.
var Feeds = map[string]Feed{
	"account-activity":          {Service: "ACCT_ACTIVITY", MaxField: 3, HasFields: true},
	"chart-equity":              {Service: "CHART_EQUITY", MaxField: 8, HasFields: true},
	"chart-futures":             {Service: "CHART_FUTURES", MaxField: 6, HasFields: true},
	"level-one-equity":          {Service: "LEVELONE_EQUITIES", MaxField: 51, HasFields: true},
	"level-one-option":          {Service: "LEVELONE_OPTIONS", MaxField: 55, HasFields: true},
	"level-one-futures":         {Service: "LEVELONE_FUTURES", MaxField: 40, HasFields: true},
	"level-one-forex":           {Service: "LEVELONE_FOREX", MaxField: 29, HasFields: true},
	"level-one-futures-options": {Service: "LEVELONE_FUTURES_OPTIONS", MaxField: 31, HasFields: true},
	"nyse-book":                 {Service: "NYSE_BOOK", MaxField: 3, HasFields: true},
	"nasdaq-book":               {Service: "NASDAQ_BOOK", MaxField: 3, HasFields: true},
	"options-book":              {Service: "OPTIONS_BOOK", MaxField: 3, HasFields: true},
	"screener-equity":           {Service: "SCREENER_EQUITY", MaxField: 4, HasFields: true},
	"screener-option":           {Service: "SCREENER_OPTION", MaxField: 4, HasFields: true},
}

// AllFieldIDs returns "0,1,...,maxField" — the fields parameter schwab-py
// sends when no explicit field list is given (all_fields()).
func AllFieldIDs(maxField int) string {
	parts := make([]string, maxField+1)
	for i := range parts {
		parts[i] = strconv.Itoa(i)
	}
	return strings.Join(parts, ",")
}

type userPreferences struct {
	StreamerInfo []struct {
		StreamerSocketURL      string `json:"streamerSocketUrl"`
		SchwabClientCustomerID string `json:"schwabClientCustomerId"`
		SchwabClientCorrelID   string `json:"schwabClientCorrelId"`
		SchwabClientChannel    string `json:"schwabClientChannel"`
		SchwabClientFunctionID string `json:"schwabClientFunctionId"`
	} `json:"streamerInfo"`
}

type request struct {
	Service                string            `json:"service"`
	RequestID              string            `json:"requestid"`
	Command                string            `json:"command"`
	SchwabClientCustomerID string            `json:"SchwabClientCustomerId"`
	SchwabClientCorrelID   string            `json:"SchwabClientCorrelId"`
	Parameters             map[string]string `json:"parameters,omitempty"`
}

type envelope struct {
	Requests []request `json:"requests"`
}

// Client is a streaming session, analogous to schwab-py's StreamClient after
// login.
type Client struct {
	rest       *client.Client
	conn       *websocket.Conn
	customerID string
	correlID   string
	channel    string
	functionID string
	requestID  atomic.Int64
	overflow   []json.RawMessage
}

// New creates an unlogged-in streaming client backed by the REST client
// (used for the user-preferences fetch and the access token).
func New(rest *client.Client) *Client {
	return &Client{rest: rest}
}

func (c *Client) makeRequest(service, command string,
	parameters map[string]string) request {
	id := c.requestID.Add(1) - 1
	return request{
		Service:                service,
		RequestID:              strconv.FormatInt(id, 10),
		Command:                command,
		SchwabClientCustomerID: c.customerID,
		SchwabClientCorrelID:   c.correlID,
		Parameters:             parameters,
	}
}

func (c *Client) send(ctx context.Context, req request) error {
	return wsjson.Write(ctx, c.conn, envelope{Requests: []request{req}})
}

// readMessage returns the next raw message, draining the overflow buffer
// first — the same ordering schwab-py's _receive guarantees.
func (c *Client) readMessage(ctx context.Context) (json.RawMessage, error) {
	if len(c.overflow) > 0 {
		msg := c.overflow[len(c.overflow)-1]
		c.overflow = c.overflow[:len(c.overflow)-1]
		return msg, nil
	}
	_, data, err := c.conn.Read(ctx)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// awaitResponse waits for the response message for one request, buffering
// any data/notify messages encountered into the overflow list, as schwab-py's
// _await_response does.
func (c *Client) awaitResponse(ctx context.Context, req request) error {
	for {
		raw, err := c.readMessage(ctx)
		if err != nil {
			return err
		}
		var msg struct {
			Response []struct {
				RequestID string `json:"requestid"`
				Service   string `json:"service"`
				Command   string `json:"command"`
				Content   struct {
					Code int    `json:"code"`
					Msg  string `json:"msg"`
				} `json:"content"`
			} `json:"response"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil || len(msg.Response) == 0 {
			c.overflow = append(c.overflow, raw)
			continue
		}
		r := msg.Response[0]
		if r.RequestID != req.RequestID || r.Service != req.Service {
			return fmt.Errorf("unexpected response: %+v", r)
		}
		if r.Content.Code != 0 {
			return fmt.Errorf("%s %s failed: code %d: %s",
				r.Service, r.Command, r.Content.Code, r.Content.Msg)
		}
		return nil
	}
}

// Login fetches user preferences, opens the websocket, and performs the
// ADMIN LOGIN handshake (schwab-py: StreamClient.login).
func (c *Client) Login(ctx context.Context) error {
	resp, err := c.rest.GetUserPreferences(ctx)
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("get_user_preferences returned status %d: %s",
			resp.StatusCode, string(resp.Body))
	}
	var prefs userPreferences
	if err := resp.JSON(&prefs); err != nil {
		return fmt.Errorf("parsing user preferences: %w", err)
	}
	if len(prefs.StreamerInfo) == 0 {
		return fmt.Errorf("user preferences contain no streamerInfo")
	}
	info := prefs.StreamerInfo[0]
	c.customerID = info.SchwabClientCustomerID
	c.correlID = info.SchwabClientCorrelID
	c.channel = info.SchwabClientChannel
	c.functionID = info.SchwabClientFunctionID

	conn, _, err := websocket.Dial(ctx, info.StreamerSocketURL, nil)
	if err != nil {
		return fmt.Errorf("connecting to streamer socket: %w", err)
	}
	c.conn = conn

	token, err := c.rest.Tokens.AccessToken(ctx)
	if err != nil {
		return err
	}
	req := c.makeRequest("ADMIN", "LOGIN", map[string]string{
		"Authorization":          token,
		"SchwabClientChannel":    c.channel,
		"SchwabClientFunctionId": c.functionID,
	})
	if err := c.send(ctx, req); err != nil {
		return err
	}
	return c.awaitResponse(ctx, req)
}

// Subscribe issues a SUBS request for one feed (schwab-py: the *_subs methods
// built on _service_op). keys are the subscription keys (symbols); for
// ACCT_ACTIVITY the key is the stream correl id, as in schwab-py. fields is
// the comma-separated field-id list; empty means all fields.
func (c *Client) Subscribe(ctx context.Context, feed Feed, keys []string,
	fields string) error {
	return c.SubscribeParams(ctx, feed, keys, fields, nil)
}

// SubscribeParams is Subscribe with additional request parameters merged in
// — for example the screener services' mandatory "sort" parameter, which
// schwab-py's screener *_subs methods pass alongside keys and fields.
func (c *Client) SubscribeParams(ctx context.Context, feed Feed, keys []string,
	fields string, extra map[string]string) error {
	if feed.Service == "ACCT_ACTIVITY" && len(keys) == 0 {
		keys = []string{c.correlID}
	}
	params := map[string]string{
		"keys": strings.Join(keys, ","),
	}
	if feed.HasFields {
		if fields == "" {
			fields = AllFieldIDs(feed.MaxField)
		}
		params["fields"] = fields
	}
	for k, v := range extra {
		params[k] = v
	}
	req := c.makeRequest(feed.Service, "SUBS", params)
	if err := c.send(ctx, req); err != nil {
		return err
	}
	return c.awaitResponse(ctx, req)
}

// Logout sends the ADMIN LOGOUT request and closes the socket
// (schwab-py: StreamClient.logout).
func (c *Client) Logout(ctx context.Context) error {
	if c.conn == nil {
		return nil
	}
	req := c.makeRequest("ADMIN", "LOGOUT", nil)
	if err := c.send(ctx, req); err == nil {
		_ = c.awaitResponse(ctx, req)
	}
	return c.conn.Close(websocket.StatusNormalClosure, "")
}

// Run subscribes and then invokes handle for every subsequent message until
// the context is cancelled or the socket fails. Responses to the
// subscription itself are consumed by Subscribe; heartbeats and data
// messages are all delivered, matching the raw message flow a schwab-py
// handler would observe.
func (c *Client) Run(ctx context.Context, feed Feed, keys []string,
	fields string, handle func(json.RawMessage) error) error {
	if err := c.Login(ctx); err != nil {
		return err
	}
	defer c.Logout(context.WithoutCancel(ctx))
	if err := c.Subscribe(ctx, feed, keys, fields); err != nil {
		return err
	}
	return c.RunLoop(ctx, handle)
}

// RunLoop delivers messages to handle until the context is cancelled or the
// socket fails. Use it after Login and one or more Subscribe calls to serve
// several feeds over one connection; Run is the single-feed convenience
// wrapper. A context cancellation is reported as a nil error.
func (c *Client) RunLoop(ctx context.Context,
	handle func(json.RawMessage) error) error {
	for {
		raw, err := c.readMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if err := handle(raw); err != nil {
			return err
		}
	}
}
