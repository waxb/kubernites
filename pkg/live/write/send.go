package write

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const maxAnswer = 8 << 20

type Answer struct {
	Code     int
	Reason   string
	Message  string
	Causes   []string
	Body     []byte
	Err      error
	NoDryRun bool
}

func (a Answer) OK() bool {
	return a.Err == nil && a.Code >= 200 && a.Code < 300
}

func (a Answer) Conflict() bool {
	return a.Err == nil && a.Code == http.StatusConflict
}

func (a Answer) Blocked() bool {
	return a.Err == nil && a.Code == http.StatusTooManyRequests
}

func (a Answer) Text() string {
	switch {
	case a.Err != nil:
		return a.Err.Error()
	case a.Message != "":
		return a.Message
	case a.Reason != "":
		return fmt.Sprintf("%d %s", a.Code, a.Reason)
	}
	return fmt.Sprintf("%d %s", a.Code, http.StatusText(a.Code))
}

type Verdict struct {
	Allowed bool
	Denied  bool
	Reason  string
	Err     error
}

var ErrUnbound = errors.New("the write gate has no connection")

func (g *Gate) do(ctx context.Context, r Request, contentType string) Answer {
	g.mu.Lock()
	client, server := g.client, g.server
	g.mu.Unlock()
	if client == nil {
		return Answer{Err: ErrUnbound}
	}
	u := server
	u.Path = strings.TrimRight(server.Path, "/") + r.Path
	u.RawPath = ""
	u.RawQuery = r.Query.Encode()
	var body io.Reader
	if r.Body != nil {
		body = bytes.NewReader(r.Body)
	}
	req, err := http.NewRequestWithContext(ctx, r.Method, u.String(), body)
	if err != nil {
		return Answer{Err: err}
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return Answer{Err: err}
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxAnswer))
	a := Answer{Code: resp.StatusCode, Body: maskedAnswer(r.Path, data), Err: err}
	if !a.OK() && err == nil {
		var st struct {
			Reason  string `json:"reason"`
			Message string `json:"message"`
			Details struct {
				Causes []struct {
					Reason  string `json:"reason"`
					Message string `json:"message"`
				} `json:"causes"`
			} `json:"details"`
		}
		if json.Unmarshal(data, &st) == nil {
			a.Reason, a.Message = st.Reason, st.Message
			for _, c := range st.Details.Causes {
				a.Causes = append(a.Causes, strings.TrimSpace(c.Reason+": "+c.Message))
			}
		}
		a.NoDryRun = r.DryRun() && a.Code == http.StatusBadRequest && strings.Contains(a.Message, "does not support dry run")
	}
	return a
}

func (g *Gate) Review(ctx context.Context, a Access) Verdict {
	body := mustJSON(map[string]any{
		"apiVersion": "authorization.k8s.io/v1",
		"kind":       "SelfSubjectAccessReview",
		"spec":       map[string]any{"resourceAttributes": a},
	})
	ans := g.do(ctx, Request{Method: http.MethodPost, Path: accessReview, Body: body}, JSONBody)
	if !ans.OK() {
		return Verdict{Err: errors.New(ans.Text())}
	}
	var review struct {
		Status struct {
			Allowed         bool   `json:"allowed"`
			Denied          bool   `json:"denied"`
			Reason          string `json:"reason"`
			EvaluationError string `json:"evaluationError"`
		} `json:"status"`
	}
	if err := json.Unmarshal(ans.Body, &review); err != nil {
		return Verdict{Err: err}
	}
	v := Verdict{Allowed: review.Status.Allowed, Denied: review.Status.Denied, Reason: review.Status.Reason}
	if v.Reason == "" && review.Status.EvaluationError != "" {
		v.Reason = review.Status.EvaluationError
	}
	return v
}

func (o Order) dryRun() Request {
	r := o.Request
	q := url.Values{}
	for k, v := range r.Query {
		q[k] = v
	}
	q["dryRun"] = []string{"All"}
	r.Query = q
	if r.Method == http.MethodDelete && len(r.Body) > 0 {
		var opts map[string]any
		if json.Unmarshal(r.Body, &opts) == nil {
			opts["dryRun"] = []string{"All"}
			r.Body = mustJSON(opts)
		}
	}
	return r
}

func (g *Gate) Preview(ctx context.Context, o Order) (Answer, []Field) {
	ans := g.do(ctx, o.dryRun(), o.ContentType)
	fields := make([]Field, len(o.Fields))
	copy(fields, o.Fields)
	if !ans.OK() {
		return ans, fields
	}
	var obj any
	if json.Unmarshal(ans.Body, &obj) == nil {
		for i := range fields {
			fields[i].After = Lookup(obj, fields[i].Path)
		}
	}
	return ans, fields
}

func (g *Gate) Send(ctx context.Context, o Order) Answer {
	return g.do(ctx, o.Request, o.ContentType)
}

func Lookup(obj any, path []string) string {
	cur := obj
	for _, step := range path {
		switch x := cur.(type) {
		case map[string]any:
			cur = x[step]
		case []any:
			key, want, ok := strings.Cut(step, "=")
			if !ok {
				i, err := strconv.Atoi(step)
				if err != nil || i < 0 || i >= len(x) {
					return ""
				}
				cur = x[i]
				continue
			}
			cur = nil
			for _, el := range x {
				if m, ok := el.(map[string]any); ok && fmt.Sprint(m[key]) == want {
					cur = m
					break
				}
			}
		default:
			return ""
		}
	}
	switch x := cur.(type) {
	case nil:
		return ""
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	}
	data, _ := json.Marshal(cur)
	return string(data)
}
