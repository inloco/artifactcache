package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/square/go-jose/v3/jwt"
)

var (
	lookup = sync.Map{}
)

type Claims struct {
	Ac  string `json:"ac,omitempty"`
	Aud string `json:"aud,omitempty"`
}

type Ac []struct {
	Scope string `json:"Scope,omitempty"`
}

func extractParams(bearer string) (httprouter.Params, error) {
	token, err := jwt.ParseSigned(bearer)
	if err != nil {
		return nil, err
	}

	var claims Claims
	if err := token.UnsafeClaimsWithoutVerification(&claims); err != nil {
		return nil, err
	}

	var ac Ac
	if err := json.Unmarshal([]byte(claims.Ac), &ac); err != nil {
		return nil, err
	}

	params := httprouter.Params{
		httprouter.Param{
			Key:   "audience",
			Value: claims.Aud,
		},
		httprouter.Param{
			Key:   "scope",
			Value: ac[0].Scope,
		},
	}

	return params, nil
}

func newRequest(baseURL string, authorization string) (*http.Request, error) {
	parsedURL, err := url.ParseRequestURI(baseURL + "_apis/artifactcache/cache?keys=e3b0c44298fc1c149afbf4c8996fb924&version=27ae41e4649b934ca495991b7852b855")
	if err != nil {
		return nil, err
	}

	request := http.Request{
		Method: http.MethodGet,
		URL:    parsedURL,
		Header: http.Header{
			"Accept": []string{
				"application/json;api-version=6.0-preview.1",
			},
			"Authorization": []string{
				authorization,
			},
		},
	}

	return &request, nil
}

func validateToken(baseURL string, authorization string) error {
	req, err := newRequest(baseURL, authorization)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if statusCode := res.StatusCode; statusCode != http.StatusOK && statusCode != http.StatusNoContent {
		return errors.New(http.StatusText(statusCode))
	}

	return nil
}

func authenticated(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		actionsCacheURL, err := base64.RawURLEncoding.DecodeString(ps.ByName("actionsCacheURL"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		authorization := r.Header.Get("Authorization")
		bearer := strings.SplitAfter(authorization, " ")[1]
		signature := strings.SplitAfter(authorization, ".")[2]

		key := fmt.Sprintf("%s#%s", actionsCacheURL, signature)
		if t, ok := lookup.Load(key); !ok || t.(time.Time).Before(time.Now().Add(-5*time.Minute)) {
			if err := validateToken(string(actionsCacheURL), authorization); err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			lookup.Store(key, time.Now())
		}

		params, err := extractParams(bearer)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		h(w, r, append(ps, params...))
	}
}
