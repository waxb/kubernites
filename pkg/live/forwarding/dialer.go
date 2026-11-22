package forwarding

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/portforward"
	"k8s.io/client-go/rest"
	clientforward "k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/client-go/transport/websocket"
	streamhttp "k8s.io/streaming/pkg/httpstream"
	streamspdy "k8s.io/streaming/pkg/httpstream/spdy"
)

const spdyPings = 5 * time.Second

type Held struct {
	mu     sync.Mutex
	closed bool
	conns  []io.Closer
	done   chan struct{}
}

func (h *Held) ended() <-chan struct{} {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.done == nil {
		h.done = make(chan struct{})
		if h.closed {
			close(h.done)
		}
	}
	return h.done
}

func (h *Held) hold(c io.Closer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		c.Close()
		return
	}
	h.conns = append(h.conns, c)
}

func (h *Held) Close() error {
	h.mu.Lock()
	if h.done != nil && !h.closed {
		close(h.done)
	}
	h.closed = true
	conns := h.conns
	h.conns = nil
	h.mu.Unlock()
	var errs []error
	for _, c := range conns {
		errs = append(errs, c.Close())
	}
	return errors.Join(errs...)
}

func Dialer(method string, u *url.URL, cfg *rest.Config, websockets bool) (streamhttp.Dialer, *Held, error) {
	held := &Held{}
	tlsConfig, err := rest.TLSConfigFor(cfg)
	if err != nil {
		return nil, nil, err
	}
	proxy := http.ProxyFromEnvironment
	if cfg.Proxy != nil {
		proxy = cfg.Proxy
	}
	base, err := streamspdy.NewRoundTripperWithConfig(streamspdy.RoundTripperConfig{TLS: tlsConfig, Proxier: proxy, PingPeriod: spdyPings})
	if err != nil {
		return nil, nil, err
	}
	upgrader := &spdyUpgrader{base: base, held: held, dialled: make(chan net.Conn, 1)}
	transport, err := rest.HTTPWrappersForConfig(cfg, upgrader)
	if err != nil {
		return nil, nil, err
	}
	dialer := spdy.NewDialerForStreaming(spdy.NewUpgraderForStreaming(upgrader), &http.Client{Transport: transport}, method, u)
	if !websockets {
		return endingDialer{Dialer: dialer, held: held}, held, nil
	}
	wsTransport, holder, err := websocket.RoundTripperFor(cfg)
	if err != nil {
		return nil, nil, err
	}
	tunneling := &tunnelingDialer{u: u, transport: wsTransport, holder: holder, held: held}
	fallback := clientforward.NewFallbackDialerForStreaming(tunneling, dialer, func(err error) bool {
		return streamhttp.IsUpgradeFailure(err) || streamhttp.IsHTTPSProxyError(err)
	})
	return endingDialer{Dialer: fallback, held: held}, held, nil
}

type endingDialer struct {
	streamhttp.Dialer
	held *Held
}

func (d endingDialer) Dial(protocols ...string) (streamhttp.Connection, string, error) {
	conn, protocol, err := d.Dialer.Dial(protocols...)
	if err != nil {
		return nil, protocol, err
	}
	return endingConnection{Connection: conn, held: d.held}, protocol, nil
}

type endingConnection struct {
	streamhttp.Connection
	held *Held
}

func (c endingConnection) CreateStream(headers http.Header) (streamhttp.Stream, error) {
	type made struct {
		stream streamhttp.Stream
		err    error
	}
	result := make(chan made, 1)
	go func() {
		stream, err := c.Connection.CreateStream(headers)
		result <- made{stream: stream, err: err}
	}()
	select {
	case m := <-result:
		return m.stream, m.err
	case <-c.Connection.CloseChan():
	case <-c.held.ended():
	}
	return nil, errors.New("the forward's connection is closed")
}

type spdyUpgrader struct {
	base    *streamspdy.SpdyRoundTripper
	held    *Held
	dialled chan net.Conn
}

func (s *spdyUpgrader) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Add(streamhttp.HeaderConnection, streamhttp.HeaderUpgrade)
	req.Header.Add(streamhttp.HeaderUpgrade, streamspdy.HeaderSpdy31)
	conn, err := s.base.Dial(req)
	if err != nil {
		return nil, err
	}
	s.held.hold(conn)
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		conn.Close()
		return nil, err
	}
	select {
	case <-s.dialled:
	default:
	}
	s.dialled <- conn
	return resp, nil
}

func (s *spdyUpgrader) NewConnection(resp *http.Response) (streamhttp.Connection, error) {
	connection := strings.ToLower(resp.Header.Get(streamhttp.HeaderConnection))
	upgrade := strings.ToLower(resp.Header.Get(streamhttp.HeaderUpgrade))
	if resp.StatusCode != http.StatusSwitchingProtocols || !strings.Contains(connection, strings.ToLower(streamhttp.HeaderUpgrade)) || !strings.Contains(upgrade, strings.ToLower(streamspdy.HeaderSpdy31)) {
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		if err != nil {
			return nil, errors.New("unable to upgrade connection: unable to read error from server response")
		}
		return nil, fmt.Errorf("unable to upgrade connection: %s", upgradeError(body))
	}
	select {
	case conn := <-s.dialled:
		return streamspdy.NewClientConnectionWithPings(conn, spdyPings)
	default:
		return nil, errors.New("unable to upgrade connection: no connection was dialled")
	}
}

func upgradeError(body []byte) string {
	var status struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if json.Unmarshal(body, &status) == nil {
		if msg := strings.TrimSpace(status.Message); msg != "" {
			return msg
		}
		if msg := strings.TrimSpace(status.Error); msg != "" {
			return msg
		}
	}
	if msg := strings.TrimSpace(string(body)); msg != "" {
		return msg
	}
	return "empty server response"
}

type tunnelingDialer struct {
	u         *url.URL
	transport http.RoundTripper
	holder    websocket.ConnectionHolder
	held      *Held
}

func (d *tunnelingDialer) Dial(protocols ...string) (streamhttp.Connection, string, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, d.u.String(), nil)
	if err != nil {
		return nil, "", err
	}
	tunneling := make([]string, 0, len(protocols))
	for _, p := range protocols {
		tunneling = append(tunneling, portforward.WebsocketsSPDYTunnelingPrefix+p)
	}
	conn, err := websocket.Negotiate(d.transport, d.holder, req, tunneling...)
	if err != nil {
		return nil, "", err
	}
	if conn == nil {
		return nil, "", errors.New("negotiated websocket connection is nil")
	}
	d.held.hold(conn)
	protocol := strings.TrimPrefix(conn.Subprotocol(), portforward.WebsocketsSPDYTunnelingPrefix)
	c, err := streamspdy.NewClientConnectionWithPings(clientforward.NewTunnelingConnection("client", conn), clientforward.PingPeriod)
	return c, protocol, err
}
