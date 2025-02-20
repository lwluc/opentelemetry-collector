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

package testcomponents // import "go.opentelemetry.io/collector/internal/testcomponents"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/model/pdata"
)

var _ config.Unmarshallable = (*ExampleExporter)(nil)

// ExampleExporter is for testing purposes. We are defining an example config and factory
// for "exampleexporter" exporter type.
type ExampleExporter struct {
	config.ExporterSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	ExtraInt                int32                    `mapstructure:"extra_int"`
	ExtraSetting            string                   `mapstructure:"extra"`
	ExtraMapSetting         map[string]string        `mapstructure:"extra_map"`
	ExtraListSetting        []string                 `mapstructure:"extra_list"`
}

// Unmarshal a config.Map data into the config struct
func (cfg *ExampleExporter) Unmarshal(componentParser *config.Map) error {
	return componentParser.UnmarshalExact(cfg)
}

const expType = "exampleexporter"

// ExampleExporterFactory is factory for ExampleExporter.
var ExampleExporterFactory = component.NewExporterFactory(
	expType,
	createExporterDefaultConfig,
	component.WithTracesExporter(createTracesExporter),
	component.WithMetricsExporter(createMetricsExporter),
	component.WithLogsExporter(createLogsExporter))

// CreateDefaultConfig creates the default configuration for the Exporter.
func createExporterDefaultConfig() config.Exporter {
	return &ExampleExporter{
		ExporterSettings: config.NewExporterSettings(config.NewComponentID(expType)),
		ExtraSetting:     "some export string",
		ExtraMapSetting:  nil,
		ExtraListSetting: nil,
	}
}

func createTracesExporter(context.Context, component.ExporterCreateSettings, config.Exporter) (component.TracesExporter, error) {
	return &ExampleExporterConsumer{}, nil
}

func createMetricsExporter(context.Context, component.ExporterCreateSettings, config.Exporter) (component.MetricsExporter, error) {
	return &ExampleExporterConsumer{}, nil
}

func createLogsExporter(context.Context, component.ExporterCreateSettings, config.Exporter) (component.LogsExporter, error) {
	return &ExampleExporterConsumer{}, nil
}

// ExampleExporterConsumer stores consumed traces and metrics for testing purposes.
type ExampleExporterConsumer struct {
	Traces           []pdata.Traces
	Metrics          []pdata.Metrics
	Logs             []pdata.Logs
	ExporterStarted  bool
	ExporterShutdown bool
}

// Start tells the exporter to start. The exporter may prepare for exporting
// by connecting to the endpoint. Host parameter can be used for communicating
// with the host after Start() has already returned.
func (exp *ExampleExporterConsumer) Start(_ context.Context, _ component.Host) error {
	exp.ExporterStarted = true
	return nil
}

// ConsumeTraces receives pdata.Traces for processing by the consumer.Traces.
func (exp *ExampleExporterConsumer) ConsumeTraces(_ context.Context, td pdata.Traces) error {
	exp.Traces = append(exp.Traces, td)
	return nil
}

func (exp *ExampleExporterConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeMetrics receives pdata.Metrics for processing by the Metrics.
func (exp *ExampleExporterConsumer) ConsumeMetrics(_ context.Context, md pdata.Metrics) error {
	exp.Metrics = append(exp.Metrics, md)
	return nil
}

func (exp *ExampleExporterConsumer) ConsumeLogs(_ context.Context, ld pdata.Logs) error {
	exp.Logs = append(exp.Logs, ld)
	return nil
}

// Shutdown is invoked during shutdown.
func (exp *ExampleExporterConsumer) Shutdown(context.Context) error {
	exp.ExporterShutdown = true
	return nil
}
