// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"testing"

	gogoproto "github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	goproto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	otlplogs "go.opentelemetry.io/collector/model/internal/data/protogen/logs/v1"
)

func TestLogRecordCount(t *testing.T) {
	logs := NewLogs()
	assert.EqualValues(t, 0, logs.LogRecordCount())

	rl := logs.ResourceLogs().AppendEmpty()
	assert.EqualValues(t, 0, logs.LogRecordCount())

	ill := rl.ScopeLogs().AppendEmpty()
	assert.EqualValues(t, 0, logs.LogRecordCount())

	ill.LogRecords().AppendEmpty()
	assert.EqualValues(t, 1, logs.LogRecordCount())

	rms := logs.ResourceLogs()
	rms.EnsureCapacity(3)
	rms.AppendEmpty().ScopeLogs().AppendEmpty()
	illl := rms.AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()
	for i := 0; i < 5; i++ {
		illl.AppendEmpty()
	}
	// 5 + 1 (from rms.At(0) initialized first)
	assert.EqualValues(t, 6, logs.LogRecordCount())
}

func TestLogRecordCountWithEmpty(t *testing.T) {
	assert.Zero(t, NewLogs().LogRecordCount())
	assert.Zero(t, Logs{orig: &otlplogs.LogsData{
		ResourceLogs: []*otlplogs.ResourceLogs{{}},
	}}.LogRecordCount())
	assert.Zero(t, Logs{orig: &otlplogs.LogsData{
		ResourceLogs: []*otlplogs.ResourceLogs{
			{
				ScopeLogs: []*otlplogs.ScopeLogs{{}},
			},
		},
	}}.LogRecordCount())
	assert.Equal(t, 1, Logs{orig: &otlplogs.LogsData{
		ResourceLogs: []*otlplogs.ResourceLogs{
			{
				ScopeLogs: []*otlplogs.ScopeLogs{
					{
						LogRecords: []*otlplogs.LogRecord{{}},
					},
				},
			},
		},
	}}.LogRecordCount())
}

func TestToFromLogProto(t *testing.T) {
	logs := LogsFromOtlp(&otlplogs.LogsData{})
	assert.EqualValues(t, NewLogs(), logs)
	assert.EqualValues(t, &otlplogs.LogsData{}, logs.orig)
}

func TestResourceLogsWireCompatibility(t *testing.T) {
	// This test verifies that OTLP ProtoBufs generated using goproto lib in
	// opentelemetry-proto repository OTLP ProtoBufs generated using gogoproto lib in
	// this repository are wire compatible.

	// Generate ResourceLogs as pdata struct.
	logs := generateTestResourceLogs()

	// Marshal its underlying ProtoBuf to wire.
	wire1, err := gogoproto.Marshal(logs.orig)
	assert.NoError(t, err)
	assert.NotNil(t, wire1)

	// Unmarshal from the wire to OTLP Protobuf in goproto's representation.
	var goprotoMessage emptypb.Empty
	err = goproto.Unmarshal(wire1, &goprotoMessage)
	assert.NoError(t, err)

	// Marshal to the wire again.
	wire2, err := goproto.Marshal(&goprotoMessage)
	assert.NoError(t, err)
	assert.NotNil(t, wire2)

	// Unmarshal from the wire into gogoproto's representation.
	var gogoprotoRS2 otlplogs.ResourceLogs
	err = gogoproto.Unmarshal(wire2, &gogoprotoRS2)
	assert.NoError(t, err)

	// Now compare that the original and final ProtoBuf messages are the same.
	// This proves that goproto and gogoproto marshaling/unmarshaling are wire compatible.
	assert.EqualValues(t, logs.orig, &gogoprotoRS2)
}

func TestLogsMoveTo(t *testing.T) {
	logs := NewLogs()
	fillTestResourceLogsSlice(logs.ResourceLogs())
	dest := NewLogs()
	logs.MoveTo(dest)
	assert.EqualValues(t, NewLogs(), logs)
	assert.EqualValues(t, generateTestResourceLogsSlice(), dest.ResourceLogs())
}

func TestLogsClone(t *testing.T) {
	logs := NewLogs()
	fillTestResourceLogsSlice(logs.ResourceLogs())
	assert.EqualValues(t, logs, logs.Clone())
}

func BenchmarkLogsClone(b *testing.B) {
	logs := NewLogs()
	fillTestResourceLogsSlice(logs.ResourceLogs())
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		clone := logs.Clone()
		if clone.ResourceLogs().Len() != logs.ResourceLogs().Len() {
			b.Fail()
		}
	}
}
