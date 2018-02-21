//  Copyright © 2018 Pixel Brewery Co. All rights reserved.

package waysense

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type httpWriter struct {
	client    *http.Client
	url       string
	apiKey    string
	apiSecret string
}

type HttpResponse struct {
	Code   int    `json:"code"`
	Result string `json:"result"`
}

const DefaultURL = "https://api-prod.pixelbrewery.co/v1/waysense/write"

// TODO might want to change this to udp
// timeout in duration form like 1s, 1m, 1h
func newHttpWriter(addr, apiKey, apiSecret, timeout string, skipSSL bool) (*httpWriter, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipSSL},
	}

	var to time.Duration
	var err error
	if timeout != "" {
		to, err = time.ParseDuration(timeout)
		if err != nil {
			to, _ = time.ParseDuration("30s")
		}
	} else {
		to, _ = time.ParseDuration("30s")
	}

	httpClient := &http.Client{
		Timeout:   to,
		Transport: tr,
	}

	apiURL := DefaultURL
	if addr != "" {
		apiURL = addr
	}

	return &httpWriter{
		client:    httpClient,
		url:       apiURL,
		apiKey:    apiKey,
		apiSecret: apiSecret,
	}, nil
}

// write to http api writer
// TODO compress body
func (w *httpWriter) Write(data []byte) (int, error) {
	var (
		err error
		req *http.Request
	)

	req, err = http.NewRequest("POST", w.url, bytes.NewReader(data))
	if err != nil {
		return 0, err
	}
	req.Header.Set("waysense.api.key", w.apiKey)
	req.Header.Set("waysense.api.secret", w.apiKey)

	resp, err := w.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, _ := ioutil.ReadAll(resp.Body)

		result := HttpResponse{}
		err = json.Unmarshal(body, &result)
		if err != nil {
			return 0, fmt.Errorf("Bad code:%d response: %s", resp.StatusCode, string(body))
		}

		return 0, nil
	} else {
		body, _ := ioutil.ReadAll(resp.Body)

		result := HttpResponse{}
		err = json.Unmarshal(body, &result)
		if err != nil {
			return 0, fmt.Errorf("Bad code:%d response: %s", resp.StatusCode, string(body))
		}

		return 0, fmt.Errorf("%s with code:%d", result.Result, result.Code)
	}
}

func (w *httpWriter) SetWriteTimeout(d time.Duration) error {
	return nil
}

func (w *httpWriter) Close() error {
	return nil
}
