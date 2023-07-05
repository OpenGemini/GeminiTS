/*
Copyright 2023 Huawei Cloud Computing Technologies Co., Ltd.

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

package executor_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/openGemini/openGemini/engine/executor"
	"github.com/openGemini/openGemini/engine/hybridqp"
	"github.com/openGemini/openGemini/open_src/influx/influxql"
	"github.com/openGemini/openGemini/open_src/influx/query"
)

func buildSortRowDataType() hybridqp.RowDataType {
	rowDataType := hybridqp.NewRowDataTypeImpl(
		influxql.VarRef{Val: "f1", Type: influxql.Integer},
		influxql.VarRef{Val: "f2", Type: influxql.String},
		influxql.VarRef{Val: "f3", Type: influxql.Float},
		influxql.VarRef{Val: "f4", Type: influxql.Boolean},
	)
	return rowDataType
}

func buildHashMergeRowDataType() hybridqp.RowDataType {
	rowDataType := hybridqp.NewRowDataTypeImpl(
		influxql.VarRef{Val: "id", Type: influxql.Integer},
		influxql.VarRef{Val: "name", Type: influxql.String},
		influxql.VarRef{Val: "value", Type: influxql.Float},
		influxql.VarRef{Val: "state", Type: influxql.Boolean},
	)
	return rowDataType
}

func BuildSortChunk1() executor.Chunk {
	rowDataType := buildSortRowDataType()

	b := executor.NewChunkBuilder(rowDataType)

	chunk := b.NewChunk("mst")

	tag1 := ParseChunkTags("tag1=1,tag2=3")
	tag2 := ParseChunkTags("tag1=2,tag2=2")
	chunk.AppendTagsAndIndex(*tag1, 0)
	chunk.AppendTagsAndIndex(*tag2, 2)

	chunk.AppendTime([]int64{1, 2, 3, 4}...)

	chunk.Column(0).AppendIntegerValues([]int64{1, 2, 3, 4}...)
	chunk.Column(0).AppendManyNotNil(4)

	chunk.Column(1).AppendStringValues([]string{"a", "b", "c", "d"}...)
	chunk.Column(1).AppendNilsV2(true, true, true, true)

	chunk.Column(2).AppendFloatValues([]float64{6.0, 5.0, 4.0, 3.0}...)
	chunk.Column(2).AppendManyNotNil(4)

	chunk.Column(3).AppendBooleanValues([]bool{true, false, true, false}...)
	chunk.Column(3).AppendManyNotNil(4)
	return chunk
}

func BuildSortChunk2() executor.Chunk {
	rowDataType := buildHashMergeRowDataType()

	b := executor.NewChunkBuilder(rowDataType)

	chunk := b.NewChunk("mst")

	tag1 := ParseChunkTags("tag1=3,tag2=1")
	chunk.AppendTagsAndIndex(*tag1, 0)

	chunk.AppendTime([]int64{5, 6}...)

	chunk.Column(0).AppendIntegerValues([]int64{5, 6}...)
	chunk.Column(0).AppendManyNotNil(2)

	chunk.Column(1).AppendStringValues([]string{"e", "f"}...)
	chunk.Column(1).AppendNilsV2(true, true)

	chunk.Column(2).AppendFloatValues([]float64{2, 1}...)
	chunk.Column(2).AppendManyNotNil(2)

	chunk.Column(3).AppendBooleanValues([]bool{true, false}...)
	chunk.Column(3).AppendManyNotNil(2)
	return chunk
}

func BuildSortChunk3() executor.Chunk {
	rowDataType := buildHashMergeRowDataType()

	b := executor.NewChunkBuilder(rowDataType)

	chunk := b.NewChunk("mst")

	tag1 := executor.ChunkTags{}
	chunk.AppendTagsAndIndex(tag1, 0)

	chunk.AppendTime([]int64{1, 2, 3, 4}...)

	chunk.Column(0).AppendIntegerValues([]int64{1, 3}...)
	chunk.Column(0).AppendNilsV2(true, false, true, false)

	chunk.Column(1).AppendStringValues([]string{"a", "b", "c"}...)
	chunk.Column(1).AppendNilsV2(true, true, true, false)

	chunk.Column(2).AppendFloatValues([]float64{6.0, 5.0, 4.0}...)
	chunk.Column(2).AppendNilsV2(true, true, true, false)

	chunk.Column(3).AppendBooleanValues([]bool{true, false, true}...)
	chunk.Column(3).AppendNilsV2(true, true, true, false)
	return chunk
}

func buildSortSchema(sortFields influxql.SortFields) *executor.QuerySchema {
	opt := query.ProcessorOptions{
		Ascending:  true,
		ChunkSize:  1024,
		Dimensions: []string{"tag1", "tag2"},
	}
	fields := make(influxql.Fields, 0)
	columnNames := make([]string, 0)
	types := []influxql.DataType{influxql.Integer, influxql.String, influxql.Float, influxql.Boolean}
	for i := 1; i <= 4; i++ {
		fields = append(fields, &influxql.Field{
			Expr: &influxql.VarRef{
				Val:   "f" + strconv.Itoa(i),
				Alias: "",
				Type:  types[i-1],
			},
		})
		columnNames = append(columnNames, "f"+strconv.Itoa(i))
	}
	return executor.NewQuerySchema(fields, columnNames, &opt, sortFields)
}

func buildSortSchema2(sortFields influxql.SortFields) *executor.QuerySchema {
	opt := query.ProcessorOptions{
		Ascending: true,
		ChunkSize: 1024,
	}
	fields := make(influxql.Fields, 0)
	columnNames := make([]string, 0)
	types := []influxql.DataType{influxql.Integer, influxql.String, influxql.Float, influxql.Boolean}
	for i := 1; i <= 4; i++ {
		fields = append(fields, &influxql.Field{
			Expr: &influxql.VarRef{
				Val:   "f" + strconv.Itoa(i),
				Alias: "",
				Type:  types[i-1],
			},
		})
		columnNames = append(columnNames, "f"+strconv.Itoa(i))
	}
	return executor.NewQuerySchema(fields, columnNames, &opt, sortFields)
}

func getResult(resultChunkOutPut *executor.ChunkPort, resultStr *string, finish chan int) {
	defer func() {
		finish <- 1
	}()
	for {
		select {
		case r, ok := <-resultChunkOutPut.State:
			if !ok {
				return
			}
			*resultStr = *resultStr + StringToRows(r)
		}
	}
}

func TestSortTransfromOrderByTagAndIntField(t *testing.T) {
	chunk1 := BuildSortChunk1()
	chunk2 := BuildSortChunk2()

	sortFields := make(influxql.SortFields, 0)
	sortFields = append(sortFields, &influxql.SortField{
		Name:      "tag2",
		Ascending: true,
	})
	sortFields = append(sortFields, &influxql.SortField{
		Name:      "f1",
		Ascending: false,
	})
	schema := buildSortSchema(sortFields)
	rt := buildSortRowDataType()
	source := NewSourceFromSingleChunk(rt, []executor.Chunk{chunk1, chunk2})
	trans, _ := executor.NewSortTransform([]hybridqp.RowDataType{rt}, []hybridqp.RowDataType{rt}, schema, schema.GetSortFields())
	var expStr string = "f1 f2 f3 f4 time\ntag1\x003\x00tag2\x001\x00\n6 f 1 false 6\n5 e 2 true 5\ntag1\x002\x00tag2\x002\x00\n4 d 3 false 4\n3 c 4 true 3\ntag1\x001\x00tag2\x003\x00\n2 b 5 false 2\n1 a 6 true 1\n"
	var resultStr string
	finish := make(chan int, 1)
	resultChunkOutPut := executor.NewChunkPort(rt)
	executor.Connect(source.Output, trans.GetInputs()[0])
	executor.Connect(trans.GetOutputs()[0], resultChunkOutPut)
	var processors executor.Processors
	processors = append(processors, source)
	processors = append(processors, trans)
	executor := executor.NewPipelineExecutor(processors)
	go getResult(resultChunkOutPut, &resultStr, finish)
	executor.Execute(context.Background())
	<-finish
	if expStr != resultStr {
		t.Error("result is error")
	}
	executor.Release()
}

func TestSortTransfromOrderByTagAndStringField(t *testing.T) {
	chunk1 := BuildSortChunk1()
	chunk2 := BuildSortChunk2()
	sortFields := make(influxql.SortFields, 0)
	sortFields = append(sortFields, &influxql.SortField{
		Name:      "tag2",
		Ascending: true,
	})
	sortFields = append(sortFields, &influxql.SortField{
		Name:      "f2",
		Ascending: false,
	})
	schema := buildSortSchema(sortFields)
	rt := buildSortRowDataType()
	source := NewSourceFromSingleChunk(rt, []executor.Chunk{chunk1, chunk2})
	trans, _ := executor.NewSortTransform([]hybridqp.RowDataType{rt}, []hybridqp.RowDataType{rt}, schema, schema.GetSortFields())
	var expStr string = "f1 f2 f3 f4 time\ntag1\x003\x00tag2\x001\x00\n6 f 1 false 6\n5 e 2 true 5\ntag1\x002\x00tag2\x002\x00\n4 d 3 false 4\n3 c 4 true 3\ntag1\x001\x00tag2\x003\x00\n2 b 5 false 2\n1 a 6 true 1\n"
	var resultStr string
	finish := make(chan int, 1)
	resultChunkOutPut := executor.NewChunkPort(rt)
	executor.Connect(source.Output, trans.GetInputs()[0])
	executor.Connect(trans.GetOutputs()[0], resultChunkOutPut)
	var processors executor.Processors
	processors = append(processors, source)
	processors = append(processors, trans)
	executor := executor.NewPipelineExecutor(processors)
	go getResult(resultChunkOutPut, &resultStr, finish)
	executor.Execute(context.Background())
	<-finish
	if expStr != resultStr {
		t.Error("result is error")
	}
	executor.Release()
}

func TestSortTransfromOrderByTagAndFloatField(t *testing.T) {
	chunk1 := BuildSortChunk1()
	chunk2 := BuildSortChunk2()
	sortFields := make(influxql.SortFields, 0)
	sortFields = append(sortFields, &influxql.SortField{
		Name:      "tag2",
		Ascending: true,
	})
	sortFields = append(sortFields, &influxql.SortField{
		Name:      "f3",
		Ascending: true,
	})
	schema := buildSortSchema(sortFields)
	rt := buildSortRowDataType()
	source := NewSourceFromSingleChunk(rt, []executor.Chunk{chunk1, chunk2})
	trans, _ := executor.NewSortTransform([]hybridqp.RowDataType{rt}, []hybridqp.RowDataType{rt}, schema, schema.GetSortFields())
	var expStr string = "f1 f2 f3 f4 time\ntag1\x003\x00tag2\x001\x00\n6 f 1 false 6\n5 e 2 true 5\ntag1\x002\x00tag2\x002\x00\n4 d 3 false 4\n3 c 4 true 3\ntag1\x001\x00tag2\x003\x00\n2 b 5 false 2\n1 a 6 true 1\n"
	var resultStr string
	finish := make(chan int, 1)
	resultChunkOutPut := executor.NewChunkPort(rt)
	executor.Connect(source.Output, trans.GetInputs()[0])
	executor.Connect(trans.GetOutputs()[0], resultChunkOutPut)
	var processors executor.Processors
	processors = append(processors, source)
	processors = append(processors, trans)
	executor := executor.NewPipelineExecutor(processors)
	go getResult(resultChunkOutPut, &resultStr, finish)
	executor.Execute(context.Background())
	<-finish
	if expStr != resultStr {
		t.Error("result is error")
	}
	fmt.Println(resultStr)
	executor.Release()
}

func TestSortTransfromOrderByTagAndBoolField(t *testing.T) {
	chunk1 := BuildSortChunk1()
	chunk2 := BuildSortChunk2()

	sortFields := make(influxql.SortFields, 0)
	sortFields = append(sortFields, &influxql.SortField{
		Name:      "tag2",
		Ascending: true,
	})
	sortFields = append(sortFields, &influxql.SortField{
		Name:      "f4",
		Ascending: true,
	})
	schema := buildSortSchema(sortFields)
	rt := buildSortRowDataType()
	source := NewSourceFromSingleChunk(rt, []executor.Chunk{chunk1, chunk2})
	trans, _ := executor.NewSortTransform([]hybridqp.RowDataType{rt}, []hybridqp.RowDataType{rt}, schema, schema.GetSortFields())
	var expStr string = "f1 f2 f3 f4 time\ntag1\x003\x00tag2\x001\x00\n6 f 1 false 6\n5 e 2 true 5\ntag1\x002\x00tag2\x002\x00\n4 d 3 false 4\n3 c 4 true 3\ntag1\x001\x00tag2\x003\x00\n2 b 5 false 2\n1 a 6 true 1\n"
	var resultStr string
	finish := make(chan int, 1)
	resultChunkOutPut := executor.NewChunkPort(rt)
	executor.Connect(source.Output, trans.GetInputs()[0])
	executor.Connect(trans.GetOutputs()[0], resultChunkOutPut)
	var processors executor.Processors
	processors = append(processors, source)
	processors = append(processors, trans)
	executor := executor.NewPipelineExecutor(processors)
	go getResult(resultChunkOutPut, &resultStr, finish)
	executor.Execute(context.Background())
	<-finish
	if expStr != resultStr {
		t.Error("result is error")
	}
	fmt.Println(resultStr)
	executor.Release()
}

func TestSortTransfromOrderByTagAndTimeField(t *testing.T) {
	chunk1 := BuildSortChunk1()
	chunk2 := BuildSortChunk2()

	sortFields := make(influxql.SortFields, 0)
	sortFields = append(sortFields, &influxql.SortField{
		Name:      "tag2",
		Ascending: true,
	})
	sortFields = append(sortFields, &influxql.SortField{
		Name:      "time",
		Ascending: false,
	})
	schema := buildSortSchema(sortFields)
	rt := buildSortRowDataType()
	source := NewSourceFromSingleChunk(rt, []executor.Chunk{chunk1, chunk2})
	trans, _ := executor.NewSortTransform([]hybridqp.RowDataType{rt}, []hybridqp.RowDataType{rt}, schema, schema.GetSortFields())
	var expStr string = "f1 f2 f3 f4 time\ntag1\x003\x00tag2\x001\x00\n6 f 1 false 6\n5 e 2 true 5\ntag1\x002\x00tag2\x002\x00\n4 d 3 false 4\n3 c 4 true 3\ntag1\x001\x00tag2\x003\x00\n2 b 5 false 2\n1 a 6 true 1\n"
	var resultStr string
	finish := make(chan int, 1)
	resultChunkOutPut := executor.NewChunkPort(rt)
	executor.Connect(source.Output, trans.GetInputs()[0])
	executor.Connect(trans.GetOutputs()[0], resultChunkOutPut)
	var processors executor.Processors
	processors = append(processors, source)
	processors = append(processors, trans)
	executor := executor.NewPipelineExecutor(processors)
	go getResult(resultChunkOutPut, &resultStr, finish)
	executor.Execute(context.Background())
	<-finish
	if expStr != resultStr {
		t.Error("result is error")
	}
	fmt.Println(resultStr)
	executor.Release()
}

func TestSortTransfromOrderByTagAndNilField(t *testing.T) {
	chunk1 := BuildSortChunk3()

	sortFields := make(influxql.SortFields, 0)
	sortFields = append(sortFields, &influxql.SortField{
		Name:      "f1",
		Ascending: false,
	})
	sortFields = append(sortFields, &influxql.SortField{
		Name:      "f2",
		Ascending: false,
	})
	schema := buildSortSchema2(sortFields)
	rt := buildSortRowDataType()
	source := NewSourceFromSingleChunk(rt, []executor.Chunk{chunk1})
	trans, _ := executor.NewSortTransform([]hybridqp.RowDataType{rt}, []hybridqp.RowDataType{rt}, schema, schema.GetSortFields())
	var expStr string = "f1 f2 f3 f4 time\n\n3 c 4 true 3\n1 a 6 true 1\n b 5 false 2\n    4\n"
	var resultStr string
	finish := make(chan int, 1)
	resultChunkOutPut := executor.NewChunkPort(rt)
	executor.Connect(source.Output, trans.GetInputs()[0])
	executor.Connect(trans.GetOutputs()[0], resultChunkOutPut)
	var processors executor.Processors
	processors = append(processors, source)
	processors = append(processors, trans)
	executor := executor.NewPipelineExecutor(processors)
	go getResult(resultChunkOutPut, &resultStr, finish)
	executor.Execute(context.Background())
	<-finish
	if expStr != resultStr {
		t.Error("result is error")
	}
	fmt.Println(resultStr)
	executor.Release()
}
