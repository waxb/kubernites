package live

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	authenticationv1 "k8s.io/api/authentication/v1"
)

const selfReviewBody = `{"apiVersion":"authentication.k8s.io/v1","kind":"SelfSubjectReview"}`

type Identity struct {
	Username string   `json:",omitempty"`
	UID      string   `json:",omitempty"`
	Groups   []string `json:",omitempty"`
	Unknown  string   `json:",omitempty"`
}

func (c *Conn) WhoAmI(ctx context.Context) (Identity, error) {
	u := c.server
	u.Path = strings.TrimRight(c.server.Path, "/") + selfReview
	u.RawPath, u.RawQuery = "", ""
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), strings.NewReader(selfReviewBody))
	if err != nil {
		return Identity{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	body, err := answer(c.client.Do(req))
	if err != nil {
		return Identity{Unknown: err.Error()}, err
	}
	var review authenticationv1.SelfSubjectReview
	if err := json.Unmarshal(body, &review); err != nil {
		return Identity{Unknown: err.Error()}, err
	}
	info := review.Status.UserInfo
	return Identity{Username: info.Username, UID: info.UID, Groups: info.Groups}, nil
}
