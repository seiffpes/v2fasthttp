package v2fasthttp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"
)

type Session struct {
	Client *Client

	BaseURL string

	Headers http.Header

	BearerToken string
	BasicUser   string
	BasicPass   string

	APIKeyHeader string
	APIKeyValue  string
}

var ErrNoClientInSession = errors.New("v2fasthttp: session has no client")

func NewSession(c *Client) *Session {
	return &Session{
		Client:  c,
		Headers: make(http.Header),
	}
}

func (s *Session) WithBaseURL(base string) *Session {
	s.BaseURL = base
	return s
}

func (s *Session) WithBearer(token string) *Session {
	s.BearerToken = token
	return s
}

func (s *Session) WithBasic(user, pass string) *Session {
	s.BasicUser = user
	s.BasicPass = pass
	return s
}

func (s *Session) WithAPIKey(header, value string) *Session {
	s.APIKeyHeader = header
	s.APIKeyValue = value
	return s
}

func (s *Session) WithHeader(key, value string) *Session {
	s.Headers.Set(key, value)
	return s
}

func (s *Session) url(p string) string {
	if s.BaseURL == "" {
		return p
	}
	u, err := url.Parse(s.BaseURL)
	if err != nil {
		return p
	}
	u.Path = path.Join(u.Path, p)
	return u.String()
}

func (s *Session) applyAuth(req *http.Request) {
	if s.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.BearerToken)
	}
	if s.BasicUser != "" || s.BasicPass != "" {
		req.SetBasicAuth(s.BasicUser, s.BasicPass)
	}
	if s.APIKeyHeader != "" && s.APIKeyValue != "" {
		req.Header.Set(s.APIKeyHeader, s.APIKeyValue)
	}
}

func (s *Session) applyHeaders(req *http.Request) {
	for k, values := range s.Headers {
		for _, v := range values {
			if req.Header.Get(k) == "" {
				req.Header.Add(k, v)
			}
		}
	}
}

func (s *Session) newRequest(ctx context.Context, method, p string, body io.Reader) (*http.Request, error) {
	u := s.url(p)
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}
	s.applyHeaders(req)
	s.applyAuth(req)
	return req, nil
}

func (s *Session) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if s.Client == nil {
		return nil, ErrNoClientInSession
	}
	return s.Client.Do(req.WithContext(ctx))
}

func (s *Session) Get(ctx context.Context, p string) (*http.Response, error) {
	req, err := s.newRequest(ctx, http.MethodGet, p, nil)
	if err != nil {
		return nil, err
	}
	return s.Do(ctx, req)
}

func (s *Session) Post(ctx context.Context, p string, body io.Reader) (*http.Response, error) {
	req, err := s.newRequest(ctx, http.MethodPost, p, body)
	if err != nil {
		return nil, err
	}
	return s.Do(ctx, req)
}

func (s *Session) Delete(ctx context.Context, p string) (*http.Response, error) {
	req, err := s.newRequest(ctx, http.MethodDelete, p, nil)
	if err != nil {
		return nil, err
	}
	return s.Do(ctx, req)
}
