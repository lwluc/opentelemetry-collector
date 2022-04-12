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

package otlpgrpc

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	v1 "go.opentelemetry.io/collector/model/internal/data/protogen/trace/v1"
	"go.opentelemetry.io/collector/model/internal/otlp"
	"go.opentelemetry.io/collector/model/pdata"
)

var _ json.Unmarshaler = TracesResponse{}
var _ json.Marshaler = TracesResponse{}

var _ json.Unmarshaler = TracesRequest{}
var _ json.Marshaler = TracesRequest{}

var tracesRequestJSON = []byte(`
	{
		"resourceSpans": [
			{
				"resource": {},
				"scopeSpans": [
					{
						"scope": {},
						"spans": [
							{
								"traceId": "",
								"spanId":"",
								"parentSpanId":"",
								"name": "test_span",
								"status": {}
							}
						]
					}
				]
			}
		]
	}`)

var tracesTransitionData = [][]byte{
	[]byte(`
	{
		"resourceSpans": [
			{
				"resource": {},
				"instrumentationLibrarySpans": [
					{
						"instrumentationLibrary": {},
						"spans": [
							{
								"traceId": "",
								"spanId":"",
								"parentSpanId":"",
								"name": "test_span",
								"status": {}
							}
						]
					}
				]
			}
		]
	}`),
	[]byte(`
	{
		"resourceSpans": [
			{
				"resource": {},
				"instrumentationLibrarySpans": [
					{
						"instrumentationLibrary": {},
						"spans": [
							{
								"traceId": "",
								"spanId":"",
								"parentSpanId":"",
								"name": "test_span",
								"status": {}
							}
						]
					}
				],
				"scopeSpans": [
					{
						"scope": {},
						"spans": [
							{
								"traceId": "",
								"spanId":"",
								"parentSpanId":"",
								"name": "test_span",
								"status": {}
							}
						]
					}
				]
			}
		]
	}`),
}

func TestTracesRequestJSON(t *testing.T) {
	tr := NewTracesRequest()
	assert.NoError(t, tr.UnmarshalJSON(tracesRequestJSON))
	assert.Equal(t, "test_span", tr.Traces().ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Name())

	got, err := tr.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, strings.Join(strings.Fields(string(tracesRequestJSON)), ""), string(got))
}

func TestTracesRequestJSONTransition(t *testing.T) {
	for _, data := range tracesTransitionData {
		tr := NewTracesRequest()
		assert.NoError(t, tr.UnmarshalJSON(data))
		assert.Equal(t, "test_span", tr.Traces().ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Name())

		got, err := tr.MarshalJSON()
		assert.NoError(t, err)
		assert.Equal(t, strings.Join(strings.Fields(string(tracesRequestJSON)), ""), string(got))
	}
}

func TestTracesGrpc(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	RegisterTracesServer(s, &fakeTracesServer{t: t})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.NoError(t, s.Serve(lis))
	}()
	t.Cleanup(func() {
		s.Stop()
		wg.Wait()
	})

	cc, err := grpc.Dial("bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, cc.Close())
	})

	logClient := NewTracesClient(cc)

	resp, err := logClient.Export(context.Background(), generateTracesRequest())
	assert.NoError(t, err)
	assert.Equal(t, NewTracesResponse(), resp)
}

func TestTracesGrpcTransition(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	RegisterTracesServer(s, &fakeTracesServer{t: t})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.NoError(t, s.Serve(lis))
	}()
	t.Cleanup(func() {
		s.Stop()
		wg.Wait()
	})

	cc, err := grpc.Dial("bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, cc.Close())
	})

	logClient := NewTracesClient(cc)

	req := generateTracesRequestWithInstrumentationLibrary()
	otlp.InstrumentationLibrarySpansToScope(req.orig.ResourceSpans)
	resp, err := logClient.Export(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, NewTracesResponse(), resp)
}

func TestTracesGrpcError(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	RegisterTracesServer(s, &fakeTracesServer{t: t, err: errors.New("my error")})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.NoError(t, s.Serve(lis))
	}()
	t.Cleanup(func() {
		s.Stop()
		wg.Wait()
	})

	cc, err := grpc.Dial("bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, cc.Close())
	})

	logClient := NewTracesClient(cc)
	resp, err := logClient.Export(context.Background(), generateTracesRequest())
	require.Error(t, err)
	st, okSt := status.FromError(err)
	require.True(t, okSt)
	assert.Equal(t, "my error", st.Message())
	assert.Equal(t, codes.Unknown, st.Code())
	assert.Equal(t, TracesResponse{}, resp)
}

type fakeTracesServer struct {
	t   *testing.T
	err error
}

func (f fakeTracesServer) Export(_ context.Context, request TracesRequest) (TracesResponse, error) {
	assert.Equal(f.t, generateTracesRequest(), request)
	return NewTracesResponse(), f.err
}

func generateTracesRequest() TracesRequest {
	td := pdata.NewTraces()
	td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("test_span")

	tr := NewTracesRequest()
	tr.SetTraces(td)
	return tr
}

func generateTracesRequestWithInstrumentationLibrary() TracesRequest {
	tr := generateTracesRequest()
	tr.orig.ResourceSpans[0].InstrumentationLibrarySpans = []*v1.InstrumentationLibrarySpans{ //nolint:staticcheck // SA1019 ignore this!
		{
			Spans: tr.orig.ResourceSpans[0].ScopeSpans[0].Spans,
		},
	}
	return tr
}
