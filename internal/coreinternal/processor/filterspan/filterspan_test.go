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

package filterspan

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/model/pdata"
	conventions "go.opentelemetry.io/collector/model/semconv/v1.5.0"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/processor/filterconfig"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/processor/filterset"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/testdata"
)

func createConfig(matchType filterset.MatchType) *filterset.Config {
	return &filterset.Config{
		MatchType: matchType,
	}
}

func TestSpan_validateMatchesConfiguration_InvalidConfig(t *testing.T) {
	testcases := []struct {
		name        string
		property    filterconfig.MatchProperties
		errorString string
	}{
		{
			name:        "empty_property",
			property:    filterconfig.MatchProperties{},
			errorString: "at least one of \"services\", \"span_names\", \"attributes\", \"libraries\" or \"resources\" field must be specified",
		},
		{
			name: "empty_service_span_names_and_attributes",
			property: filterconfig.MatchProperties{
				Services: []string{},
			},
			errorString: "at least one of \"services\", \"span_names\", \"attributes\", \"libraries\" or \"resources\" field must be specified",
		},
		{
			name: "log_properties",
			property: filterconfig.MatchProperties{
				LogNames: []string{"log"},
			},
			errorString: "log_names should not be specified for trace spans",
		},
		{
			name: "invalid_match_type",
			property: filterconfig.MatchProperties{
				Config:   *createConfig("wrong_match_type"),
				Services: []string{"abc"},
			},
			errorString: "error creating service name filters: unrecognized match_type: 'wrong_match_type', valid types are: [regexp strict]",
		},
		{
			name: "missing_match_type",
			property: filterconfig.MatchProperties{
				Services: []string{"abc"},
			},
			errorString: "error creating service name filters: unrecognized match_type: '', valid types are: [regexp strict]",
		},
		{
			name: "invalid_regexp_pattern_service",
			property: filterconfig.MatchProperties{
				Config:   *createConfig(filterset.Regexp),
				Services: []string{"["},
			},
			errorString: "error creating service name filters: error parsing regexp: missing closing ]: `[`",
		},
		{
			name: "invalid_regexp_pattern_span",
			property: filterconfig.MatchProperties{
				Config:    *createConfig(filterset.Regexp),
				SpanNames: []string{"["},
			},
			errorString: "error creating span name filters: error parsing regexp: missing closing ]: `[`",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := NewMatcher(&tc.property)
			assert.Nil(t, output)
			assert.EqualError(t, err, tc.errorString)
		})
	}
}

func TestSpan_Matching_False(t *testing.T) {
	testcases := []struct {
		name       string
		properties *filterconfig.MatchProperties
	}{
		{
			name: "service_name_doesnt_match_regexp",
			properties: &filterconfig.MatchProperties{
				Config:     *createConfig(filterset.Regexp),
				Services:   []string{"svcA"},
				Attributes: []filterconfig.Attribute{},
			},
		},

		{
			name: "service_name_doesnt_match_strict",
			properties: &filterconfig.MatchProperties{
				Config:     *createConfig(filterset.Strict),
				Services:   []string{"svcA"},
				Attributes: []filterconfig.Attribute{},
			},
		},

		{
			name: "span_name_doesnt_match",
			properties: &filterconfig.MatchProperties{
				Config:     *createConfig(filterset.Regexp),
				SpanNames:  []string{"spanNo.*Name"},
				Attributes: []filterconfig.Attribute{},
			},
		},

		{
			name: "span_name_doesnt_match_any",
			properties: &filterconfig.MatchProperties{
				Config: *createConfig(filterset.Regexp),
				SpanNames: []string{
					"spanNo.*Name",
					"non-matching?pattern",
					"regular string",
				},
				Attributes: []filterconfig.Attribute{},
			},
		},
	}

	span := pdata.NewSpan()
	span.SetName("spanName")
	library := pdata.NewInstrumentationLibrary()
	resource := pdata.NewResource()

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			matcher, err := NewMatcher(tc.properties)
			require.NoError(t, err)
			assert.NotNil(t, matcher)

			assert.False(t, matcher.MatchSpan(span, resource, library))
		})
	}
}

func TestSpan_MissingServiceName(t *testing.T) {
	cfg := &filterconfig.MatchProperties{
		Config:   *createConfig(filterset.Regexp),
		Services: []string{"svcA"},
	}

	mp, err := NewMatcher(cfg)
	assert.Nil(t, err)
	assert.NotNil(t, mp)

	emptySpan := pdata.NewSpan()
	assert.False(t, mp.MatchSpan(emptySpan, pdata.NewResource(), pdata.NewInstrumentationLibrary()))
}

func TestSpan_Matching_True(t *testing.T) {
	testcases := []struct {
		name       string
		properties *filterconfig.MatchProperties
	}{
		{
			name: "service_name_match_regexp",
			properties: &filterconfig.MatchProperties{
				Config:     *createConfig(filterset.Regexp),
				Services:   []string{"svcA"},
				Attributes: []filterconfig.Attribute{},
			},
		},
		{
			name: "service_name_match_strict",
			properties: &filterconfig.MatchProperties{
				Config:     *createConfig(filterset.Strict),
				Services:   []string{"svcA"},
				Attributes: []filterconfig.Attribute{},
			},
		},
		{
			name: "span_name_match",
			properties: &filterconfig.MatchProperties{
				Config:     *createConfig(filterset.Regexp),
				SpanNames:  []string{"span.*"},
				Attributes: []filterconfig.Attribute{},
			},
		},
		{
			name: "span_name_second_match",
			properties: &filterconfig.MatchProperties{
				Config: *createConfig(filterset.Regexp),
				SpanNames: []string{
					"wrong.*pattern",
					"span.*",
					"yet another?pattern",
					"regularstring",
				},
				Attributes: []filterconfig.Attribute{},
			},
		},
	}

	span := pdata.NewSpan()
	span.SetName("spanName")
	span.Attributes().InsertString("keyString", "arithmetic")
	span.Attributes().InsertInt("keyInt", 123)
	span.Attributes().InsertDouble("keyDouble", 3245.6)
	span.Attributes().InsertBool("keyBool", true)
	span.Attributes().InsertString("keyExists", "present")
	assert.NotNil(t, span)

	resource := pdata.NewResource()
	resource.Attributes().InsertString(conventions.AttributeServiceName, "svcA")

	library := pdata.NewInstrumentationLibrary()

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			mp, err := NewMatcher(tc.properties)
			require.NoError(t, err)
			assert.NotNil(t, mp)

			assert.True(t, mp.MatchSpan(span, resource, library))
		})
	}
}

func TestServiceNameForResource(t *testing.T) {
	td := testdata.GenerateTracesOneSpanNoResource()
	name, found := serviceNameForResource(td.ResourceSpans().At(0).Resource())
	require.Equal(t, name, "<nil-service-name>")
	require.False(t, found)

	td = testdata.GenerateTracesOneSpan()
	resource := td.ResourceSpans().At(0).Resource()
	name, found = serviceNameForResource(resource)
	require.Equal(t, name, "<nil-service-name>")
	require.False(t, found)

	resource.Attributes().InsertString(conventions.AttributeServiceName, "test-service")
	name, found = serviceNameForResource(resource)
	require.Equal(t, name, "test-service")
	require.True(t, found)
}

func TestServiceNameForSpan(t *testing.T) {
	td := testdata.GenerateTracesOneSpanNoResource()
	span := td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0)
	name, found := serviceNameForSpan(span)
	require.Equal(t, name, "<nil-service-name>")
	require.False(t, found)

	span.Attributes().InsertString(conventions.AttributeServiceName, "test-service")
	name, found = serviceNameForSpan(span)
	require.Equal(t, name, "test-service")
	require.True(t, found)
}
