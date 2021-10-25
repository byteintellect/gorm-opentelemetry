// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otelgorm

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/semconv/v1.4.0"
	otelTrace "go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
	"strings"
)

const (
	spanName = "gorm.query"
	dbOperationKey = semconv.DBOperationKey
	dbStatementKey = semconv.DBStatementKey
)

func dbTable(name string) attribute.KeyValue {
	return attribute.String("db.table", name)
}

func dbStatement(stmt string) attribute.KeyValue {
	return dbStatementKey.String(stmt)
}

func dbCount(n int64) attribute.KeyValue {
	return attribute.Int64("db.count", n)
}

func dbOperation(op string) attribute.KeyValue {
	return dbOperationKey.String(op)
}

func (op *OtelPlugin) before(tx *gorm.DB) {
	tx.Statement.Context, _ = op.tracer.
		Start(tx.Statement.Context, spanName, otelTrace.WithSpanKind(otelTrace.SpanKindClient))
}

func extractQuery(tx *gorm.DB) string {
	return tx.Dialector.Explain(tx.Statement.SQL.String(), tx.Statement.Vars...)
}

func (op *OtelPlugin) after(operation string) gormHookFunc {
	return func(tx *gorm.DB) {
		span := otelTrace.SpanFromContext(tx.Statement.Context)
		if !span.IsRecording() {
			// skip the reporting if not recording
			return
		}
		defer span.End()

		// Error
		if tx.Error != nil {
			span.SetStatus(codes.Error, tx.Error.Error())
		}

		// extract the db operation
		query := extractQuery(tx)
		if operation == "" {
			operation = strings.ToUpper(strings.Split(query, " ")[0])
		}

		if tx.Statement.Table != "" {
			span.SetAttributes(dbTable(tx.Statement.Table))
		}

		span.SetAttributes(
			dbStatement(query),
			dbOperation(operation),
			dbCount(tx.Statement.RowsAffected),
		)
	}
}
