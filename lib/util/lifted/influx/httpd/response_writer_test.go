/*
Copyright 2024 Huawei Cloud Computing Technologies Co., Ltd.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package httpd

import (
	"testing"

	"github.com/influxdata/influxdb/models"
	"github.com/openGemini/openGemini/lib/util/lifted/influx/query"
)

type mockWriter struct {
}

func (w *mockWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func BenchmarkJsonFormatter(b *testing.B) {
	f := &jsonFormatter{}
	res := []*query.Result{}
	for i := 0; i < 100; i++ {
		res = append(res, &query.Result{Series: []*models.Row{{
			Name:    "db2",
			Columns: []string{"name", "query"},
			Values: [][]interface{}{
				{"db2_query_name", "db2_query"},
				{"db2_query2_name", "db2_query2"},
			},
		},
			{
				Name:    "db4",
				Columns: []string{"name", "query"},
				Values: [][]interface{}{
					{"db4_query_name", "db4_query"},
					{"db4_query2_name", "db4_query2"},
					{"db4_query3_name", "db4_query3"},
				},
			}}})
	}
	resp := Response{Results: res}
	writer := &mockWriter{}
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		f.WriteResponse(writer, resp)
	}
}

func BenchmarkSonicJsonFormatter(b *testing.B) {
	f := &sonicJsonFormatter{}
	res := []*query.Result{}
	for i := 0; i < 100; i++ {
		res = append(res, &query.Result{Series: []*models.Row{{
			Name:    "db2",
			Columns: []string{"name", "query"},
			Values: [][]interface{}{
				{"db2_query_name", "db2_query"},
				{"db2_query2_name", "db2_query2"},
			},
		},
			{
				Name:    "db4",
				Columns: []string{"name", "query"},
				Values: [][]interface{}{
					{"db4_query_name", "db4_query"},
					{"db4_query2_name", "db4_query2"},
					{"db4_query3_name", "db4_query3"},
				},
			}}})
	}
	resp := Response{Results: res}
	writer := &mockWriter{}
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		f.WriteResponse(writer, resp)
	}
}
