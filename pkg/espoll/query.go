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

package espoll

import "encoding/json"

type BoolQuery struct {
	Filter             []any
	Must               []any
	MustNot            []any
	Should             []any
	MinimumShouldMatch int
	Boost              float64
}

func (q BoolQuery) MarshalJSON() ([]byte, error) {
	type boolQuery struct {
		Filter             []any   `json:"filter,omitempty"`
		Must               []any   `json:"must,omitempty"`
		MustNot            []any   `json:"must_not,omitempty"`
		Should             []any   `json:"should,omitempty"`
		MinimumShouldMatch int     `json:"minimum_should_match,omitempty"`
		Boost              float64 `json:"boost,omitempty"`
	}
	return encodeQueryJSON("bool", boolQuery(q))
}

type ExistsQuery struct {
	Field string
}

func (q ExistsQuery) MarshalJSON() ([]byte, error) {
	return encodeQueryJSON("exists", map[string]any{
		"field": q.Field,
	})
}

type TermQuery struct {
	Field string
	Value any
	Boost float64
}

func (q TermQuery) MarshalJSON() ([]byte, error) {
	type termQuery struct {
		Value any     `json:"value"`
		Boost float64 `json:"boost,omitempty"`
	}
	return encodeQueryJSON("term", map[string]any{
		q.Field: termQuery{q.Value, q.Boost},
	})
}

type TermsQuery struct {
	Field  string
	Values []any
	Boost  float64
}

func (q TermsQuery) MarshalJSON() ([]byte, error) {
	args := map[string]any{
		q.Field: q.Values,
	}
	if q.Boost != 0 {
		args["boost"] = q.Boost
	}
	return encodeQueryJSON("terms", args)
}

type MatchPhraseQuery struct {
	Field string
	Value any
}

func (q MatchPhraseQuery) MarshalJSON() ([]byte, error) {
	return encodeQueryJSON("match_phrase", map[string]any{
		q.Field: q.Value,
	})
}

func encodeQueryJSON(k string, v any) ([]byte, error) {
	m := map[string]any{k: v}
	return json.Marshal(m)
}
