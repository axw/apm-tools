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

package approvaltest

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/tidwall/gjson"

	"github.com/elastic/apm-tools/pkg/espoll"
)

// ApproveEvents compares the _source of the search hits with the
// contents of the file in "systemtest/approvals/<name>.approved.json".
//
// Dynamic fields (@timestamp, observer.id, etc.) are replaced
// with a static string for comparison. Integration tests elsewhere
// use canned data to test fields that we do not cover here.
//
// If the events differ, then the test will fail.
func ApproveEvents(t testing.TB, name string, hits []espoll.SearchHit, dynamic ...string) {
	t.Helper()

	// Fields generated by the server (e.g. observer.*)
	// agent which may change between tests.
	//
	// Ignore their values in comparisons, but compare
	// existence: either the field exists in both, or neither.
	dynamic = append([]string{
		"ecs.version",
		"event.ingested",
		"observer.ephemeral_id",
		"observer.hostname",
		"observer.id",
		"observer.version",
	}, dynamic...)

	// Sort events for repeatable diffs.
	sort.Sort(apmEventSearchHits(hits))

	sources := make([][]byte, len(hits))
	for i, hit := range hits {
		sources[i] = hit.RawSource
	}
	ApproveEventDocs(t, filepath.Join("approvals", name), sources, dynamic...)
}

var apmEventSortFields = []string{
	"processor.event",
	"trace.id",
	"transaction.id",
	"span.id",
	"error.id",
	"transaction.name",
	"span.destination.service.resource",
	"transaction.type",
	"span.type",
	"service.name",
	"service.environment",
	"message",
	"metricset.interval", // useful to sort different interval metric sets.
	"@timestamp",         // last resort before _id; order is generally guaranteed
}

type apmEventSearchHits []espoll.SearchHit

func (hits apmEventSearchHits) Len() int {
	return len(hits)
}

func (hits apmEventSearchHits) Swap(i, j int) {
	hits[i], hits[j] = hits[j], hits[i]
}

func (hits apmEventSearchHits) Less(i, j int) bool {
	for _, field := range apmEventSortFields {
		ri := gjson.GetBytes(hits[i].RawSource, field)
		rj := gjson.GetBytes(hits[j].RawSource, field)
		if ri.Less(rj, true) {
			return true
		}
		if rj.Less(ri, true) {
			return false
		}
	}
	// All _source fields are equivalent, so compare doc _ids.
	return hits[i].ID < hits[j].ID
}
