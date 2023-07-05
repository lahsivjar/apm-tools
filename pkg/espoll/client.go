// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Package espoll wraps an elasticsearch.Client to provide utilitarian methods
// to assert for conditions until a certain threshold is met.
package espoll

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// Client wraps an Elasticsearch client
type Client struct {
	*elasticsearch.Client
}

// WrapClient wraps an Elasticsearch client and returns an espoll.Client
func WrapClient(c *elasticsearch.Client) *Client { return &Client{Client: c} }

type Request interface {
	Do(ctx context.Context, transport esapi.Transport) (*esapi.Response, error)
}

// Do performs the specified request.
func (es *Client) Do(
	ctx context.Context,
	req Request,
	out any,
	opts ...RequestOption,
) (*esapi.Response, error) {
	requestOptions := requestOptions{
		// Set the timeout to something high to account for Elasticsearch
		// cluster and index/shard initialisation. Under normal conditions
		// this timeout should never be reached.
		timeout:  time.Minute,
		interval: 100 * time.Millisecond,
	}
	for _, opt := range opts {
		opt(&requestOptions)
	}
	var timeoutC, tickerC <-chan time.Time
	var transport esapi.Transport = es
	if requestOptions.cond != nil {
		// A return condition has been specified, which means we
		// might retry the request. Wrap the transport with a
		// bodyRepeater to ensure the body is copied as needed.
		transport = &bodyRepeater{transport, nil}
		if requestOptions.timeout > 0 {
			timeout := time.NewTimer(requestOptions.timeout)
			defer timeout.Stop()
			timeoutC = timeout.C
		}
	}

	var resp *esapi.Response
	for {
		if tickerC != nil {
			select {
			case <-timeoutC:
				return nil, context.DeadlineExceeded
			case <-tickerC:
			}
		}
		var err error
		resp, err = req.Do(ctx, transport)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.IsError() {
			return nil, &Error{StatusCode: resp.StatusCode, Message: resp.String()}
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		if out != nil {
			if err := json.Unmarshal(body, out); err != nil {
				return nil, err
			}
		}
		resp.Body = io.NopCloser(bytes.NewReader(body))
		if requestOptions.cond == nil || requestOptions.cond(resp) {
			break
		}
		if tickerC == nil {
			// First time around, start a ticker for retrying.
			ticker := time.NewTicker(requestOptions.interval)
			defer ticker.Stop()
			tickerC = ticker.C
		}
	}
	return resp, nil
}

// RequestOption modifies certain parameters for an esapi.Request.
type RequestOption func(*requestOptions)

type Error struct {
	StatusCode int
	Message    string
}

func (e *Error) Error() string {
	return e.Message
}

type requestOptions struct {
	timeout  time.Duration
	interval time.Duration
	cond     ConditionFunc
}

// WithTimeout sets the timeout in an Elasticsearch request.
func WithTimeout(d time.Duration) RequestOption {
	return func(opts *requestOptions) {
		opts.timeout = d
	}
}

// WithInterval sets the poll interval in an Elasticsearch request.
func WithInterval(d time.Duration) RequestOption {
	return func(opts *requestOptions) {
		opts.interval = d
	}
}

// ConditionFunc evaluates the esapi.Response.
type ConditionFunc func(*esapi.Response) bool

// AllCondition returns a ConditionFunc that returns true as
// long as none of the supplied conditions returns false.
func AllCondition(conds ...ConditionFunc) ConditionFunc {
	return func(resp *esapi.Response) bool {
		for _, cond := range conds {
			if !cond(resp) {
				return false
			}
		}
		return true
	}
}

// WithCondition runs the specified condition func.
func WithCondition(cond ConditionFunc) RequestOption {
	return func(opts *requestOptions) {
		opts.cond = cond
	}
}
