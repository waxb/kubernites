package livetest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/util/httpstream/wsstream"
	streamhttp "k8s.io/streaming/pkg/httpstream"
	streamspdy "k8s.io/streaming/pkg/httpstream/spdy"
)

type Shell struct {
	mu       sync.Mutex
	urls     []string
	typed    bytes.Buffer
	sizes    []string
	closed   bool
	Tar      []byte
	Dirs     []string
	Replay   map[string]string
	uploaded bytes.Buffer
}

func (f *Shell) Uploaded() []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return bytes.Clone(f.uploaded.Bytes())
}

const success = `{"metadata":{},"status":"Success"}`

func (f *Shell) replay(command []string, stdin io.Reader, stdout io.Writer) bool {
	f.mu.Lock()
	out, ok := "", false
	if len(command) > 0 {
		out, ok = f.Replay[command[0]]
	}
	f.mu.Unlock()
	if !ok {
		return false
	}
	io.WriteString(stdout, out)
	buf := make([]byte, 256)
	for {
		n, err := stdin.Read(buf)
		if n > 0 {
			f.record(func() { f.typed.Write(buf[:n]) })
		}
		if err != nil {
			f.record(func() { f.closed = true })
			return true
		}
	}
}

func (f *Shell) tar(command []string, stdin io.Reader, stdout, status io.Writer) bool {
	if len(command) == 0 {
		return false
	}
	switch {
	case command[0] == "test" && len(command) == 3 && command[1] == "-d":
		f.mu.Lock()
		dir := slices.Contains(f.Dirs, command[2])
		f.mu.Unlock()
		if dir {
			io.WriteString(status, success)
		} else {
			io.WriteString(status, `{"metadata":{},"status":"Failure","message":"command terminated with non-zero exit code: 1","reason":"NonZeroExitCode","details":{"causes":[{"reason":"ExitCode","message":"1"}]}}`)
		}
		return true
	case command[0] == "tar" && len(command) >= 3 && command[1] == "cf":
		f.mu.Lock()
		data := bytes.Clone(f.Tar)
		f.mu.Unlock()
		stdout.Write(data)
		io.WriteString(status, success)
		return true
	case command[0] == "tar" && slices.Contains(command, "-xmf"):
		data, _ := io.ReadAll(stdin)
		f.record(func() { f.uploaded.Write(data) })
		io.WriteString(status, success)
		return true
	}
	return false
}

func (f *Shell) record(fn func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fn()
}

func (f *Shell) Serve(w http.ResponseWriter, r *http.Request) {
	f.record(func() { f.urls = append(f.urls, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery) })
	if !wsstream.IsWebSocketRequest(r) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	channels := []wsstream.ChannelType{wsstream.ReadChannel, wsstream.WriteChannel, wsstream.WriteChannel, wsstream.WriteChannel, wsstream.ReadChannel}
	conn := wsstream.NewConn(map[string]wsstream.ChannelProtocolConfig{
		"v5.channel.k8s.io": {Binary: true, Channels: channels},
		"v4.channel.k8s.io": {Binary: true, Channels: channels},
	})
	_, streams, err := conn.Open(w, r)
	if err != nil {
		return
	}
	defer conn.Close()
	stdin, stdout, status, resize := streams[0], streams[1], streams[3], streams[4]
	if f.tar(r.URL.Query()["command"], stdin, stdout, status) {
		return
	}
	if f.replay(r.URL.Query()["command"], stdin, stdout) {
		return
	}
	if r.URL.Query().Get("tty") == "true" {
		go func() {
			dec := json.NewDecoder(resize)
			for {
				var size struct{ Width, Height uint16 }
				if dec.Decode(&size) != nil {
					return
				}
				f.record(func() { f.sizes = append(f.sizes, fmt.Sprintf("%dx%d", size.Width, size.Height)) })
				fmt.Fprintf(stdout, "\x1b[2J\x1b[Hsize %dx%d\r\n", size.Width, size.Height)
			}
		}()
	}
	fmt.Fprint(stdout, "$ ")
	buf := make([]byte, 256)
	line := ""
	for {
		n, err := stdin.Read(buf)
		if n > 0 {
			f.record(func() { f.typed.Write(buf[:n]) })
			stdout.Write(buf[:n])
			line += string(buf[:n])
			if strings.HasSuffix(line, "\r") {
				cmd := strings.TrimSpace(line)
				line = ""
				if code, ok := strings.CutPrefix(cmd, "exit "); ok {
					fmt.Fprintf(status, `{"metadata":{},"status":"Failure","message":"command terminated with non-zero exit code: %s","reason":"NonZeroExitCode","details":{"causes":[{"reason":"ExitCode","message":%q}]}}`, code, code)
					return
				}
				if cmd == "exit" {
					fmt.Fprint(status, `{"metadata":{},"status":"Success"}`)
					return
				}
				fmt.Fprintf(stdout, "\r\nran %s\r\n$ ", cmd)
			}
		}
		if err != nil {
			f.record(func() { f.closed = true })
			return
		}
	}
}

func (f *Shell) Seen() (string, []string, []string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.typed.String(), append([]string(nil), f.urls...), append([]string(nil), f.sizes...), f.closed
}

func (f *Shell) ServeForward(w http.ResponseWriter, r *http.Request) {
	f.record(func() { f.urls = append(f.urls, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery) })
	if wsstream.IsWebSocketRequest(r) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, err := streamhttp.Handshake(r, w, []string{"portforward.k8s.io"}); err != nil {
		return
	}
	streams := make(chan streamhttp.Stream, 8)
	conn := streamspdy.NewResponseUpgrader().UpgradeResponse(w, r, func(s streamhttp.Stream, _ <-chan struct{}) error {
		streams <- s
		return nil
	})
	if conn == nil {
		return
	}
	defer conn.Close()
	for {
		select {
		case s := <-streams:
			if s.Headers().Get("streamType") != "data" {
				continue
			}
			port := s.Headers().Get("port")
			go func() {
				defer s.Close()
				buf := make([]byte, 1024)
				for {
					n, err := s.Read(buf)
					if n > 0 {
						fmt.Fprintf(s, "port %s heard %s", port, buf[:n])
					}
					if err != nil {
						return
					}
				}
			}()
		case <-conn.CloseChan():
			f.record(func() { f.closed = true })
			return
		}
	}
}
