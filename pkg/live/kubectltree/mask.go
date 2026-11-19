package kubectltree

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	Masked     = "****"
	lastConfig = "kubectl.kubernetes.io/last-applied-configuration"
)

var MaskedData = base64.StdEncoding.EncodeToString([]byte(Masked))

func underSecrets(req *http.Request) bool {
	p := req.URL.Path
	return strings.Contains(p, "/api/") && strings.Contains(p+"/", "/secrets/")
}

func preferJSON(req *http.Request) *http.Request {
	accept := req.Header.Get("Accept")
	if !strings.Contains(accept, "protobuf") && !strings.Contains(accept, "cbor") {
		return req
	}
	var keep []string
	for _, part := range strings.Split(accept, ",") {
		if !strings.Contains(part, "protobuf") && !strings.Contains(part, "cbor") {
			keep = append(keep, part)
		}
	}
	out := req.Clone(req.Context())
	if len(keep) == 0 {
		keep = []string{"application/json"}
	}
	out.Header.Set("Accept", strings.Join(keep, ","))
	return out
}

func maskResponse(resp *http.Response, watch bool) *http.Response {
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return resp
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "json") {
		resp.Body.Close()
		resp.Body = io.NopCloser(strings.NewReader(""))
		resp.StatusCode, resp.Status = http.StatusNotAcceptable, "406 Not Acceptable"
		return resp
	}
	if watch {
		pr, pw := io.Pipe()
		body := resp.Body
		go func() {
			dec := json.NewDecoder(body)
			enc := json.NewEncoder(pw)
			for {
				var ev any
				if err := dec.Decode(&ev); err != nil {
					body.Close()
					if errors.Is(err, io.EOF) {
						pw.Close()
					} else {
						pw.CloseWithError(err)
					}
					return
				}
				if err := enc.Encode(maskValue(ev)); err != nil {
					body.Close()
					return
				}
			}
		}()
		resp.Body = pr
		resp.ContentLength = -1
		resp.Header.Del("Content-Length")
		return resp
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	var v any
	if err != nil || json.Unmarshal(data, &v) != nil {
		resp.Body = io.NopCloser(strings.NewReader(""))
		resp.StatusCode, resp.Status = http.StatusBadGateway, "502 Bad Gateway"
		return resp
	}
	out, _ := json.Marshal(maskValue(v))
	resp.Body = io.NopCloser(bytes.NewReader(out))
	resp.ContentLength = int64(len(out))
	resp.Header.Set("Content-Length", fmt.Sprint(len(out)))
	return resp
}

func maskValue(v any) any {
	m, ok := v.(map[string]any)
	if !ok {
		return v
	}
	switch m["kind"] {
	case "Secret":
		maskSecret(m)
	case "Table":
		if rows, ok := m["rows"].([]any); ok {
			for _, row := range rows {
				if r, ok := row.(map[string]any); ok {
					r["object"] = maskValue(r["object"])
					maskAnnotation(r["object"])
				}
			}
		}
	}
	if items, ok := m["items"].([]any); ok {
		for i, it := range items {
			items[i] = maskValue(it)
			if im, ok := it.(map[string]any); ok && im["kind"] == nil {
				maskSecret(im)
			}
		}
	}
	if obj, ok := m["object"]; ok && m["type"] != nil {
		m["object"] = maskValue(obj)
	}
	return m
}

func maskSecret(m map[string]any) {
	if data, ok := m["data"].(map[string]any); ok {
		for k := range data {
			data[k] = MaskedData
		}
	}
	if data, ok := m["stringData"].(map[string]any); ok {
		for k := range data {
			data[k] = Masked
		}
	}
	maskAnnotation(m)
}

func maskAnnotation(v any) {
	m, ok := v.(map[string]any)
	if !ok {
		return
	}
	meta, _ := m["metadata"].(map[string]any)
	if ann, ok := meta["annotations"].(map[string]any); ok {
		if _, has := ann[lastConfig]; has {
			ann[lastConfig] = Masked
		}
	}
}

func (t *Tree) Reveal(ctx context.Context, namespace, name, key string) (string, error) {
	args := []string{"get", "secret", name, "-o", "json"}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	res := t.runAs(ctx, args, hooks{reveal: true})
	if res.Code != 0 {
		return "", errors.New(strings.TrimSpace(res.Output))
	}
	var s struct {
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal([]byte(res.Stdout), &s); err != nil {
		return "", err
	}
	v, ok := s.Data[key]
	if !ok {
		return "", fmt.Errorf("the Secret %s has no key %q", name, key)
	}
	return v, nil
}
