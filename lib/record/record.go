/*
Copyright 2022 Huawei Cloud Computing Technologies Co., Ltd.

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

// nolint
package record

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/openGemini/openGemini/lib/cpu"
	"github.com/openGemini/openGemini/lib/statisticsPusher/statistics"
	"github.com/openGemini/openGemini/open_src/vm/protoparser/influx"
)

const (
	TimeField           = "time"
	RecMaxLenForRuse    = 512
	RecMaxRowNumForRuse = 2024
)

var intervalRecAppendFunctions map[int]func(rec, iRec *Record, index, row int)
var intervalRecUpdateFunctions map[int]func(rec, iRec *Record, index, row, recRow int)
var recTransAppendFunctions map[int]func(rec, iRec *Record, index, row int)

func init() {
	intervalRecAppendFunctions = make(map[int]func(rec, iRec *Record, index, row int))
	recTransAppendFunctions = make(map[int]func(rec, iRec *Record, index, row int))
	intervalRecUpdateFunctions = make(map[int]func(rec, iRec *Record, index, row, recRow int))

	intervalRecAppendFunctions[influx.Field_Type_String] = stringAppendFunction
	intervalRecAppendFunctions[influx.Field_Type_Tag] = stringAppendFunction
	intervalRecAppendFunctions[influx.Field_Type_Int] = integerAppendFunction
	intervalRecAppendFunctions[influx.Field_Type_Float] = floatAppendFunction
	intervalRecAppendFunctions[influx.Field_Type_Boolean] = booleanAppendFunction

	intervalRecUpdateFunctions[influx.Field_Type_String] = stringUpdateFunction
	intervalRecUpdateFunctions[influx.Field_Type_Tag] = stringUpdateFunction
	intervalRecUpdateFunctions[influx.Field_Type_Int] = integerUpdateFunction
	intervalRecUpdateFunctions[influx.Field_Type_Float] = floatUpdateFunction
	intervalRecUpdateFunctions[influx.Field_Type_Boolean] = booleanUpdateFunction

	recTransAppendFunctions[influx.Field_Type_String] = recStringAppendFunction
	recTransAppendFunctions[influx.Field_Type_Tag] = recStringAppendFunction
	recTransAppendFunctions[influx.Field_Type_Int] = recIntegerAppendFunction
	recTransAppendFunctions[influx.Field_Type_Float] = recFloatAppendFunction
	recTransAppendFunctions[influx.Field_Type_Boolean] = recBooleanAppendFunction
}

type Record struct {
	*RecMeta
	ColVals []ColVal
	Schema  Schemas
}

type RecMeta struct {
	IntervalIndex []int
	// Times used to store the time for
	// first/last aggregation
	Times    [][]int64
	tagIndex []int
	tags     []*[]byte
	ColMeta  []ColMeta // used for pre agg
}

func (r *RecMeta) IsEmpty() bool {
	for _, meta := range r.ColMeta {
		if !meta.IsEmpty() {
			return false
		}
	}
	return true
}

func (r *RecMeta) AssignRecMetaTimes(t [][]int64) {
	r.Times = t
}

func (r *RecMeta) Copy() *RecMeta {
	copyMeta := &RecMeta{}
	copyMeta.IntervalIndex = make([]int, len(r.IntervalIndex))
	copyMeta.tagIndex = make([]int, len(r.tagIndex))
	copyMeta.ColMeta = make([]ColMeta, len(r.ColMeta))
	copyMeta.tags = make([]*[]byte, len(r.tags))
	copyMeta.Times = make([][]int64, len(r.Times))

	copy(copyMeta.IntervalIndex, r.IntervalIndex)
	copy(copyMeta.tagIndex, r.tagIndex)
	for index, colM := range r.ColMeta {
		val, err := colM.Clone()
		if err != nil {
			copyMeta.ColMeta[index] = ColMeta{}
		} else {
			copyMeta.ColMeta[index] = val
		}

	}

	for index, tag := range r.tags {
		copyMeta.tags[index] = cloneBytes(*tag)
	}

	for i, times := range r.Times {
		copyMeta.Times[i] = make([]int64, len(times))
		copy(copyMeta.Times[i], times)
	}

	return copyMeta
}

func (r *RecMeta) ResetDeep() {
	for i := range r.ColMeta {
		r.ColMeta[i].Init()
	}
	r.tags = r.tags[:0]
	r.IntervalIndex = r.IntervalIndex[:0]
	r.tagIndex = r.tagIndex[:0]
	r.ColMeta = r.ColMeta[:0]
	r.Times = r.Times[:0]
}
func (r *RecMeta) ResetForReuse() {
	for i := range r.ColMeta {
		r.ColMeta[i].Init()
	}
	r.tags = r.tags[:0]
	r.tagIndex = r.tagIndex[:0]
	r.IntervalIndex = r.IntervalIndex[:0]
	for i := range r.Times {
		r.Times[i] = r.Times[i][:0]
	}
}

func cloneBytes(v []byte) *[]byte {
	var clone = make([]byte, len(v))
	copy(clone, v)
	return &clone
}

type ColMeta struct {
	isSetFlag bool // check whether ColMeta has been set value. default false.

	min     interface{}
	max     interface{}
	minTime int64
	maxTime int64

	first     interface{}
	last      interface{}
	firstTime int64
	lastTime  int64

	sum   interface{}
	count interface{}
}

func (m *ColMeta) Init() {
	m.isSetFlag = false

	m.min = nil
	m.max = nil
	m.minTime = 0
	m.maxTime = 0

	m.first = nil
	m.last = nil
	m.firstTime = 0
	m.lastTime = 0

	m.sum = nil
	m.count = nil
}

func (m *ColMeta) IsEmpty() bool {
	return !m.isSetFlag
}

func (m *ColMeta) Min() (interface{}, int64) {
	return m.min, m.minTime
}

func (m *ColMeta) Clone() (ColMeta, error) {
	if m == nil {
		return ColMeta{}, nil
	}
	var clone ColMeta

	clone.isSetFlag = m.isSetFlag
	clone.min = m.min
	clone.max = m.max
	clone.minTime = m.minTime
	clone.maxTime = m.maxTime

	clone.first = m.first
	clone.last = m.last
	clone.firstTime = m.firstTime
	clone.lastTime = m.lastTime

	clone.sum = m.sum
	clone.count = m.count
	return clone, nil
}

func (m *ColMeta) Max() (interface{}, int64) {
	return m.max, m.maxTime
}

func (m *ColMeta) First() (interface{}, int64) {
	return m.first, m.firstTime
}

func (m *ColMeta) Last() (interface{}, int64) {
	return m.last, m.lastTime
}

func (m *ColMeta) Sum() interface{} {
	return m.sum
}

func (m *ColMeta) Count() interface{} {
	return m.count
}

func (m *ColMeta) SetMin(min interface{}, minTime int64) {
	m.isSetFlag = true
	m.min = min
	m.minTime = minTime
}

func (m *ColMeta) SetMax(max interface{}, maxTime int64) {
	m.isSetFlag = true
	m.max = max
	m.maxTime = maxTime
}

func (m *ColMeta) SetFirst(first interface{}, firstTime int64) {
	m.isSetFlag = true
	m.first = first
	m.firstTime = firstTime
}

func (m *ColMeta) SetLast(last interface{}, lastTime int64) {
	m.isSetFlag = true
	m.last = last
	m.lastTime = lastTime
}

func (m *ColMeta) SetSum(sum interface{}) {
	m.isSetFlag = true
	m.sum = sum
}

func (m *ColMeta) SetCount(count interface{}) {
	m.isSetFlag = true
	m.count = count
}

type SortAux struct {
	RowIds  []int32
	Times   []int64
	SortRec *Record
}

func (aux *SortAux) Len() int {
	return len(aux.RowIds)
}

func (aux *SortAux) Less(i, j int) bool {
	return aux.Times[i] < aux.Times[j]
}

func (aux *SortAux) Swap(i, j int) {
	aux.Times[i], aux.Times[j] = aux.Times[j], aux.Times[i]
	aux.RowIds[i], aux.RowIds[j] = aux.RowIds[j], aux.RowIds[i]
}

func (aux *SortAux) Init(times []int64) {
	aux.init(times)
}

func (aux *SortAux) init(times []int64) {
	size := len(times)
	if cap(aux.Times) < size {
		aux.Times = make([]int64, size)
	}
	aux.Times = aux.Times[:size]

	if cap(aux.RowIds) < size {
		aux.RowIds = make([]int32, size)
	}
	aux.RowIds = aux.RowIds[:size]

	for i := 0; i < size; i++ {
		aux.RowIds[i] = int32(i)
		aux.Times[i] = times[i]
	}
}

func (aux *SortAux) InitRecord(schemas Schemas) {
	if aux.SortRec == nil {
		aux.SortRec = NewRecordBuilder(schemas)
	} else {
		aux.SortRec.ResetWithSchema(schemas)
	}
}

func NewRecord(schema Schemas, initColMeta bool) *Record {
	schemaLen := schema.Len()
	record := &Record{}
	record.Schema = append(record.Schema, make([]Field, schemaLen)...)
	record.Schema = record.Schema[:schemaLen]
	record.ColVals = append(record.ColVals, make([]ColVal, schemaLen)...)
	record.ColVals = record.ColVals[:schemaLen]
	copy(record.Schema, schema)
	if initColMeta {
		record.RecMeta = &RecMeta{}
		record.ColMeta = append(record.ColMeta, make([]ColMeta, schemaLen)...)
		record.ColMeta = record.ColMeta[:schemaLen]
	}
	return record
}

func (rec Record) Len() int {
	return len(rec.Schema)
}

func (rec Record) Swap(i, j int) {
	rec.Schema[i], rec.Schema[j] = rec.Schema[j], rec.Schema[i]
	rec.ColVals[i], rec.ColVals[j] = rec.ColVals[j], rec.ColVals[i]
}

func (rec Record) Less(i, j int) bool {
	if rec.Schema[i].Name == TimeField {
		return false
	} else if rec.Schema[j].Name == TimeField {
		return true
	} else {
		return rec.Schema[i].Name < rec.Schema[j].Name
	}
}

func (rec *Record) ReserveColVal(size int) {
	// resize col val
	colLen := len(rec.ColVals)
	colCap := cap(rec.ColVals)
	remain := colCap - colLen
	if delta := size - remain; delta > 0 {
		rec.ColVals = append(rec.ColVals[:colCap], make([]ColVal, delta)...)
	}
	rec.ColVals = rec.ColVals[:colLen+size]
	rec.InitColVal(colLen, colLen+size)
}

func (rec *Record) ReserveSchema(size int) {
	// resize col val
	colLen := len(rec.Schema)
	colCap := cap(rec.Schema)
	remain := colCap - colLen
	if delta := size - remain; delta > 0 {
		rec.Schema = append(rec.Schema[:colCap], make([]Field, delta)...)
	}
	rec.Schema = rec.Schema[:colLen+size]
}

func (rec *Record) FieldIndexs(colName string) int {
	for i := range rec.Schema {
		if rec.Schema[i].Name == colName {
			return i
		}
	}
	return -1
}

func (rec *Record) InitColVal(start, end int) {
	for i := start; i < end; i++ {
		cv := &rec.ColVals[i]
		cv.Init()
	}
}

func (rec *Record) ReserveSchemaAndColVal(size int) {
	if size > 0 {
		rec.ReserveSchema(size)
		rec.ReserveColVal(size)
	}
}

func (rec *Record) ColumnAppendNull(colIdx int) {
	if rec.Schema[colIdx].Type == influx.Field_Type_Int {
		rec.ColVals[colIdx].AppendIntegerNull()
	} else if rec.Schema[colIdx].Type == influx.Field_Type_Float {
		rec.ColVals[colIdx].AppendFloatNull()
	} else if rec.Schema[colIdx].Type == influx.Field_Type_Boolean {
		rec.ColVals[colIdx].AppendBooleanNull()
	} else if rec.Schema[colIdx].Type == influx.Field_Type_String {
		rec.ColVals[colIdx].AppendStringNull()
	} else {
		panic("error type")
	}
}

func (rec *Record) CopyWithCondition(ascending bool, tr TimeRange, schema Schemas) *Record {
	times := rec.Times()
	startIndex := GetTimeRangeStartIndex(times, 0, tr.Min)
	endIndex := GetTimeRangeEndIndex(times, 0, tr.Max)

	if startIndex <= endIndex {
		copyRec := Record{}
		copyRec.SetSchema(schema)
		copyRec.ReserveColVal(len(schema))
		isExist := false
		if ascending {
			for i := 0; i < len(schema)-1; i++ {
				colIndex := rec.FieldIndexs(schema[i].Name)
				if colIndex >= 0 {
					isExist = true
					copyRec.ColVals[i].AppendColVal(&rec.ColVals[colIndex], rec.Schema[colIndex].Type, startIndex, endIndex+1)
				} else {
					copyRec.ColVals[i].PadColVal(copyRec.Schema[i].Type, endIndex-startIndex+1)
				}
			}
			if isExist {
				// append time column
				timeIndex := rec.ColNums() - 1
				copyRec.ColVals[len(schema)-1].AppendColVal(&rec.ColVals[timeIndex], rec.Schema[timeIndex].Type, startIndex, endIndex+1)
				return &copyRec
			}
			return nil
		}

		for i := 0; i < len(schema)-1; i++ {
			colIndex := rec.FieldIndexs(schema[i].Name)
			if colIndex < 0 {
				copyRec.ColVals[i].PadColVal(copyRec.Schema[i].Type, endIndex-startIndex+1)
				continue
			}
			isExist = true
			for pos := endIndex; pos >= startIndex; pos-- {
				copyRec.ColVals[i].AppendColVal(&rec.ColVals[colIndex], rec.Schema[colIndex].Type, pos, pos+1)
			}
		}
		if isExist {
			// append time column
			timeIndex := rec.ColNums() - 1
			for pos := endIndex; pos >= startIndex; pos-- {
				copyRec.ColVals[len(schema)-1].AppendColVal(&rec.ColVals[timeIndex], rec.Schema[timeIndex].Type, pos, pos+1)
			}
			return &copyRec
		}
		return nil
	}
	return nil
}

func (rec *Record) Copy() *Record {
	copyRec := &Record{}
	copyRec.CopyImpl(rec, true, true)
	return copyRec
}

func (rec *Record) CopyImpl(srcRec *Record, setSchema, reserveColVal bool) {
	times := srcRec.Times()
	startIndex := 0
	endIndex := len(times) - 1
	if startIndex <= endIndex {
		if srcRec.RecMeta != nil {
			rec.RecMeta = srcRec.RecMeta.Copy()
		}
		if setSchema {
			rec.SetSchema(srcRec.Schema)
		}
		if reserveColVal {
			rec.ReserveColVal(len(srcRec.Schema))
		}
		isExist := false
		for i := 0; i < len(srcRec.Schema)-1; i++ {
			colIndex := srcRec.FieldIndexs(srcRec.Schema[i].Name)
			if colIndex >= 0 {
				isExist = true
				rec.ColVals[i].AppendColVal(&srcRec.ColVals[colIndex], srcRec.Schema[colIndex].Type, startIndex, endIndex+1)
			} else {
				rec.ColVals[i].PadColVal(rec.Schema[i].Type, endIndex-startIndex+1)
			}
		}
		if isExist {
			// append time column
			timeIndex := srcRec.ColNums() - 1
			rec.ColVals[len(srcRec.Schema)-1].AppendColVal(&srcRec.ColVals[timeIndex], srcRec.Schema[timeIndex].Type, startIndex, endIndex+1)
			return
		}
		return
	}
}

func (rec *Record) Clone() *Record {

	clone := NewRecordBuilder(rec.Schema)
	for i := 0; i < len(rec.Schema)-1; i++ {
		clone.ColVals[i].AppendAll(&rec.ColVals[i])
	}
	return clone
}

func (rec *Record) CopyColVals() []ColVal {
	times := rec.Times()
	startIndex := 0
	endIndex := len(times) - 1
	if startIndex <= endIndex {
		var colVals []ColVal
		colVals = append(colVals, make([]ColVal, len(rec.Schema))...)
		for _, v := range colVals {
			v.Init()
		}
		isExist := false
		for i := 0; i < len(rec.Schema)-1; i++ {
			colIndex := rec.FieldIndexs(rec.Schema[i].Name)
			isExist = true
			colVals[i].AppendColVal(&rec.ColVals[colIndex], rec.Schema[colIndex].Type, startIndex, endIndex+1)
		}
		if isExist {
			// append time column
			timeIndex := rec.ColNums() - 1
			colVals[len(rec.Schema)-1].AppendColVal(&rec.ColVals[timeIndex], rec.Schema[timeIndex].Type, startIndex, endIndex+1)
			return colVals
		}
		return nil
	}
	return nil
}

func (rec *Record) TimeColumn() *ColVal {
	return &rec.ColVals[len(rec.Schema)-1]
}

func (rec *Record) Schemas() []Field {
	return rec.Schema
}

func (rec *Record) Columns() []ColVal {
	return rec.ColVals
}

func (rec *Record) RowNums() int {
	if rec == nil || len(rec.ColVals) == 0 {
		return 0
	}
	return rec.ColVals[len(rec.ColVals)-1].Len
}

func (rec *Record) ColNums() int {
	return len(rec.ColVals)
}

func (rec *Record) Column(i int) *ColVal {
	return &rec.ColVals[i]
}

func (rec *Record) SetSchema(schemas Schemas) {
	rec.Schema = rec.Schema[:0]
	rec.Schema = append(rec.Schema, schemas...)
}

func (rec *Record) MinTime(isAscending bool) int64 {
	if isAscending {
		return rec.firstTime()
	}
	return rec.lastTime()
}

func (rec *Record) MaxTime(isAscending bool) int64 {
	if isAscending {
		return rec.lastTime()
	}
	return rec.firstTime()
}

func (rec *Record) firstTime() int64 {
	timeCol := &rec.ColVals[len(rec.ColVals)-1]
	return timeCol.IntegerValues()[0]
}

func (rec *Record) lastTime() int64 {
	timeCol := &rec.ColVals[len(rec.ColVals)-1]
	return timeCol.IntegerValues()[timeCol.Len-1]
}

func (rec *Record) Time(i int) int64 {
	timeCol := &rec.ColVals[len(rec.ColVals)-1]
	return timeCol.IntegerValues()[i]
}

func (rec *Record) AppendTime(time ...int64) {
	for _, t := range time {
		rec.ColVals[len(rec.ColVals)-1].AppendInteger(t)
	}
}

func (rec *Record) mergeRecordNonOverlap(newRec, oldRec *Record, newPos, oldPos, newRows, oldRows, limitRows int) (int, int) {
	rec.mergeRecordSchema(newRec, oldRec)
	// resize record col val
	mergeRecLen := len(rec.ColVals)
	mergeRecCap := cap(rec.ColVals)
	remain := mergeRecCap - mergeRecLen
	if len(rec.Schema) > remain {
		rec.ColVals = make([]ColVal, len(rec.Schema))
	}
	rec.ColVals = rec.ColVals[:len(rec.Schema)]

	// exclude time column
	newSchemaLen := len(newRec.Schema) - 1
	oldSchemaLen := len(oldRec.Schema) - 1

	var newEnd, oldEnd int
	if oldRows-oldPos < limitRows {
		oldEnd = oldRows
		limitRows -= oldRows - oldPos
		if newRows-newPos <= limitRows {
			newEnd = newRows
		} else {
			newEnd = newPos + limitRows
		}
	} else {
		oldEnd = oldPos + limitRows
		newEnd = newPos
	}

	iNew, iOld, idx := 0, 0, 0
	for {
		if iNew < newSchemaLen && iOld < oldSchemaLen {
			if oldRec.Schema[iOld].Name < newRec.Schema[iNew].Name {
				rec.ColVals[idx].AppendColVal(&oldRec.ColVals[iOld], oldRec.Schema[iOld].Type, oldPos, oldEnd)
				rec.ColVals[idx].PadColVal(oldRec.Schema[iOld].Type, newEnd-newPos)
				iOld++
			} else if newRec.Schema[iNew].Name < oldRec.Schema[iOld].Name {
				rec.ColVals[idx].PadColVal(newRec.Schema[iNew].Type, oldEnd-oldPos)
				rec.ColVals[idx].AppendColVal(&newRec.ColVals[iNew], newRec.Schema[iNew].Type, newPos, newEnd)
				iNew++
			} else {
				rec.ColVals[idx].AppendColVal(&oldRec.ColVals[iOld], oldRec.Schema[iOld].Type, oldPos, oldEnd)
				rec.ColVals[idx].AppendColVal(&newRec.ColVals[iNew], newRec.Schema[iNew].Type, newPos, newEnd)
				iNew++
				iOld++
			}
			idx++
			continue
		}

		if iNew < newSchemaLen {
			for iNew < newSchemaLen {
				rec.ColVals[idx].PadColVal(newRec.Schema[iNew].Type, oldEnd-oldPos)
				rec.ColVals[idx].AppendColVal(&newRec.ColVals[iNew], newRec.Schema[iNew].Type, newPos, newEnd)
				iNew++
				idx++
			}
		} else if iOld < oldSchemaLen {
			for iOld < oldSchemaLen {
				rec.ColVals[idx].AppendColVal(&oldRec.ColVals[iOld], oldRec.Schema[iOld].Type, oldPos, oldEnd)
				rec.ColVals[idx].PadColVal(oldRec.Schema[iOld].Type, newEnd-newPos)
				iOld++
				idx++
			}
		}
		break
	}

	// append time col
	rec.ColVals[idx].AppendColVal(&oldRec.ColVals[iOld], oldRec.Schema[iOld].Type, oldPos, oldEnd)
	rec.ColVals[idx].AppendColVal(&newRec.ColVals[iNew], newRec.Schema[iNew].Type, newPos, newEnd)

	return newEnd, oldEnd
}

func (rec *Record) mergeRecordSchema(newRec, oldRec *Record) {
	iNew, iOld := 0, 0
	newSchemaLen, oldSchemaLen := len(newRec.Schema)-1, len(oldRec.Schema)-1
	for {
		if iNew < newSchemaLen && iOld < oldSchemaLen {
			if newRec.Schema[iNew].Name < oldRec.Schema[iOld].Name {
				rec.Schema = append(rec.Schema, Field{newRec.Schema[iNew].Type, newRec.Schema[iNew].Name})
				iNew++
			} else if oldRec.Schema[iOld].Name < newRec.Schema[iNew].Name {
				rec.Schema = append(rec.Schema, Field{oldRec.Schema[iOld].Type, oldRec.Schema[iOld].Name})
				iOld++
			} else {
				rec.Schema = append(rec.Schema, Field{newRec.Schema[iNew].Type, newRec.Schema[iNew].Name})
				iNew++
				iOld++
			}
			continue
		}
		// include time col
		if iNew < newSchemaLen {
			rec.Schema = append(rec.Schema, newRec.Schema[iNew:]...)
		} else if iOld < oldSchemaLen {
			rec.Schema = append(rec.Schema, oldRec.Schema[iOld:]...)
		} else {
			rec.Schema = append(rec.Schema, newRec.Schema[newSchemaLen])
		}
		return
	}
}

func (rec *Record) mergeRecRow(newRec, oldRec *Record, newRowIdx, oldRowIdx int) {
	iNew, iOld, idx := 0, 0, 0
	newRecLen, oldRecLen := len(newRec.Schema)-1, len(oldRec.Schema)-1
	for iNew < newRecLen && iOld < oldRecLen {
		if newRec.Schema[iNew].Name < oldRec.Schema[iOld].Name {
			rec.ColVals[idx].AppendColVal(&newRec.ColVals[iNew], newRec.Schema[iNew].Type, newRowIdx, newRowIdx+1)
			iNew++
		} else if newRec.Schema[iNew].Name > oldRec.Schema[iOld].Name {
			rec.ColVals[idx].AppendColVal(&oldRec.ColVals[iOld], oldRec.Schema[iOld].Type, oldRowIdx, oldRowIdx+1)
			iOld++
		} else {
			if !newRec.ColVals[iNew].IsNil(newRowIdx) {
				rec.ColVals[idx].AppendColVal(&newRec.ColVals[iNew], newRec.Schema[iNew].Type, newRowIdx, newRowIdx+1)
			} else if !oldRec.ColVals[iOld].IsNil(oldRowIdx) {
				rec.ColVals[idx].AppendColVal(&oldRec.ColVals[iOld], oldRec.Schema[iOld].Type, oldRowIdx, oldRowIdx+1)
			} else {
				rec.ColVals[idx].PadColVal(newRec.Schema[iNew].Type, 1)
			}
			iNew++
			iOld++
		}
		idx++
	}

	for iOld < oldRecLen {
		rec.ColVals[idx].AppendColVal(&oldRec.ColVals[iOld], oldRec.Schema[iOld].Type, oldRowIdx, oldRowIdx+1)
		iOld++
		idx++
	}
	for iNew < newRecLen {
		rec.ColVals[idx].AppendColVal(&newRec.ColVals[iNew], newRec.Schema[iNew].Type, newRowIdx, newRowIdx+1)
		iNew++
		idx++
	}

	// append time col
	rec.ColVals[idx].AppendColVal(&newRec.ColVals[iNew], rec.Schema[idx].Type, newRowIdx, newRowIdx+1)
}

func (rec *Record) AppendRec(srcRec *Record, start, end int) {
	rec.appendRecImpl(srcRec, start, end, true)
}

func (rec *Record) appendRecImpl(srcRec *Record, start, end int, pad bool) {
	if start == end {
		return
	}

	iRec, iSrcRec := 0, 0
	recLen, srcRecLen := len(rec.Schema)-1, len(srcRec.Schema)-1
	for {
		if iRec < recLen && iSrcRec < srcRecLen {
			// srcRec.Name < rec.Name is not exist
			if srcRec.Schema[iSrcRec].Name > rec.Schema[iRec].Name {
				rec.ColVals[iRec].PadColVal(rec.Schema[iRec].Type, end-start)
			} else {
				rec.ColVals[iRec].AppendColVal(&srcRec.ColVals[iSrcRec], srcRec.Schema[iSrcRec].Type, start, end)
				iSrcRec++
			}
			iRec++
			continue
		}

		if pad {
			for iRec < recLen {
				rec.ColVals[iRec].PadColVal(rec.Schema[iRec].Type, end-start)
				iRec++
			}
		}
		break
	}
	// append time col
	rec.ColVals[recLen].AppendColVal(&srcRec.ColVals[srcRecLen], srcRec.Schema[srcRecLen].Type, start, end)

	// append RecMeta
	if srcRec.RecMeta != nil && len(srcRec.RecMeta.Times) > 0 {
		if rec.RecMeta == nil {
			rec.RecMeta = &RecMeta{
				Times: make([][]int64, srcRecLen+1),
			}
		}
		for i, t := range srcRec.RecMeta.Times {
			if len(t) != 0 {
				rec.RecMeta.Times[i] = append(rec.RecMeta.Times[i], t[start:end]...)
			}
		}
	}
}

func (rec *Record) AppendRecForTagSet(srcRec *Record, start, end int) {
	rec.appendRecImpl(srcRec, start, end, false)
}

func (rec *Record) AppendRecForSeries(srcRec *Record, start, end int, ridIdx map[int]struct{}) {
	// note: there is not RecMeta to deal with.
	if start == end {
		return
	}

	var idx int
	for i := range srcRec.Schema {
		if _, ok := ridIdx[i]; ok {
			continue
		}
		rec.ColVals[idx].AppendColVal(&srcRec.ColVals[i], srcRec.Schema[i].Type, start, end)
		idx++
	}
}

func (rec *Record) appendRecs(ascending bool, newRec, oldRec *Record, newStart, newEnd, oldStart, oldEnd int,
	newTimeVals, oldTimeVals []int64, limitRows int) (int, int, int) {
	if ascending {
		for newStart < newEnd && oldStart < oldEnd {
			if oldTimeVals[oldStart] < newTimeVals[newStart] {
				rec.AppendRec(oldRec, oldStart, oldStart+1)
				oldStart++
			} else if newTimeVals[newStart] < oldTimeVals[oldStart] {
				rec.AppendRec(newRec, newStart, newStart+1)
				newStart++
			} else {
				rec.mergeRecRow(newRec, oldRec, newStart, oldStart)
				newStart++
				oldStart++
			}
			limitRows--
			if limitRows == 0 {
				return 0, newStart, oldStart
			}
		}
	} else {
		for newStart < newEnd && oldStart < oldEnd {
			if oldTimeVals[oldStart] < newTimeVals[newStart] {
				rec.AppendRec(newRec, newStart, newStart+1)
				newStart++
			} else if newTimeVals[newStart] < oldTimeVals[oldStart] {
				rec.AppendRec(oldRec, oldStart, oldStart+1)
				oldStart++
			} else {
				rec.mergeRecRow(newRec, oldRec, newStart, oldStart)
				newStart++
				oldStart++
			}
			limitRows--
			if limitRows == 0 {
				return 0, newStart, oldStart
			}
		}
	}

	if newStart < newEnd {
		curNewRows := newEnd - newStart
		if curNewRows >= limitRows {
			rec.AppendRec(newRec, newStart, newStart+limitRows)
			return 0, newStart + limitRows, oldStart
		}
		rec.AppendRec(newRec, newStart, newEnd)
		limitRows -= curNewRows
	} else if oldStart < oldEnd {
		curOldRows := oldEnd - oldStart
		if curOldRows >= limitRows {
			rec.AppendRec(oldRec, oldStart, oldStart+limitRows)
			return 0, newStart, oldStart + limitRows
		}
		rec.AppendRec(oldRec, oldStart, oldEnd)
		limitRows -= curOldRows
	}
	return limitRows, newEnd, oldEnd
}

func (rec *Record) mergeRecordOverlapImpl(newRec, oldRec *Record, newOpStart, newOpEnd, oldOpStart, oldOpEnd int, newTimeVals, oldTimeVals []int64,
	newPos, oldPos, newRows, oldRows, limitRows int, ascending bool) (int, int) {
	if newOpEnd > newRows {
		newOpEnd = newRows
	}
	if oldOpEnd > oldRows {
		oldOpEnd = oldRows
	}

	if oldOpStart == oldPos {
		curNewRows := newOpStart - newPos
		if curNewRows >= limitRows {
			rec.AppendRec(newRec, newPos, newPos+limitRows)
			return newPos + limitRows, oldPos
		}
		rec.AppendRec(newRec, newPos, newOpStart)
		limitRows -= curNewRows
	} else {
		curOldRows := oldOpStart - oldPos
		if curOldRows >= limitRows {
			rec.AppendRec(oldRec, oldPos, oldPos+limitRows)
			return newPos, oldPos + limitRows
		}
		rec.AppendRec(oldRec, oldPos, oldOpStart)
		limitRows -= curOldRows
	}

	var newEnd, oldEnd int
	limitRows, newEnd, oldEnd = rec.appendRecs(ascending, newRec, oldRec, newOpStart, newOpEnd, oldOpStart, oldOpEnd,
		newTimeVals, oldTimeVals, limitRows)
	if limitRows == 0 {
		return newEnd, oldEnd
	}

	if oldEnd == oldRows {
		if newRows-newEnd >= limitRows {
			rec.AppendRec(newRec, newEnd, newEnd+limitRows)
			return newEnd + limitRows, oldEnd
		}
		rec.AppendRec(newRec, newEnd, newRows)
		return newRows, oldRows
	}

	if oldRows-oldEnd >= limitRows {
		rec.AppendRec(oldRec, oldEnd, oldEnd+limitRows)
		return newEnd, oldEnd + limitRows
	}
	rec.AppendRec(oldRec, oldEnd, oldRows)
	return newRows, oldRows
}

func (rec *Record) mergeRecordOverlap(newRec, oldRec *Record, newTimeVals, oldTimeVals []int64, newPos, oldPos, newRows, oldRows, limitRows int) (int, int) {
	rec.mergeRecordSchema(newRec, oldRec)
	// resize record col val
	mergeRecLen := len(rec.ColVals)
	mergeRecCap := cap(rec.ColVals)
	remain := mergeRecCap - mergeRecLen
	if len(rec.Schema) > remain {
		rec.ColVals = make([]ColVal, len(rec.Schema))
	}
	rec.ColVals = rec.ColVals[:len(rec.Schema)]

	var newEnd, oldEnd int
	if newTimeVals[newPos] < oldTimeVals[oldPos] {
		if newTimeVals[newRows-1] <= oldTimeVals[oldRows-1] {
			newEnd, oldEnd = rec.mergeRecordOverlapImpl(newRec, oldRec,
				GetTimeRangeStartIndex(newTimeVals, newPos, oldTimeVals[oldPos]), newRows,
				oldPos, GetTimeRangeStartIndex(oldTimeVals, oldPos, newTimeVals[newRows-1])+1,
				newTimeVals, oldTimeVals, newPos, oldPos, newRows, oldRows, limitRows, true)
		} else {
			newEnd, oldEnd = rec.mergeRecordOverlapImpl(newRec, oldRec,
				GetTimeRangeStartIndex(newTimeVals, newPos, oldTimeVals[oldPos]), GetTimeRangeStartIndex(newTimeVals, newPos, oldTimeVals[oldRows-1])+1,
				oldPos, oldRows,
				newTimeVals, oldTimeVals, newPos, oldPos, newRows, oldRows, limitRows, true)
		}
	} else {
		if newTimeVals[newRows-1] <= oldTimeVals[oldRows-1] {
			newEnd, oldEnd = rec.mergeRecordOverlapImpl(newRec, oldRec,
				newPos, newRows,
				GetTimeRangeStartIndex(oldTimeVals, oldPos, newTimeVals[newPos]), GetTimeRangeStartIndex(oldTimeVals, oldPos, newTimeVals[newRows-1]+1),
				newTimeVals, oldTimeVals, newPos, oldPos, newRows, oldRows, limitRows, true)
		} else {
			newEnd, oldEnd = rec.mergeRecordOverlapImpl(newRec, oldRec,
				newPos, GetTimeRangeStartIndex(newTimeVals, newPos, oldTimeVals[oldRows-1])+1,
				GetTimeRangeStartIndex(oldTimeVals, oldPos, newTimeVals[newPos]), oldRows,
				newTimeVals, oldTimeVals, newPos, oldPos, newRows, oldRows, limitRows, true)
		}
	}

	return newEnd, oldEnd
}

func (rec *Record) mergeRecordOverlapDescend(newRec, oldRec *Record, newTimeVals, oldTimeVals []int64, newPos, oldPos, newRows, oldRows, limitRows int) (int, int) {
	rec.mergeRecordSchema(newRec, oldRec)
	// resize record col val
	mergeRecLen := len(rec.ColVals)
	mergeRecCap := cap(rec.ColVals)
	remain := mergeRecCap - mergeRecLen
	if len(rec.Schema) > remain {
		rec.ColVals = make([]ColVal, len(rec.Schema))
	}
	rec.ColVals = rec.ColVals[:len(rec.Schema)]

	var newEnd, oldEnd int
	if newTimeVals[newRows-1] < oldTimeVals[oldRows-1] {
		if newTimeVals[newPos] <= oldTimeVals[oldPos] {
			newEnd, oldEnd = rec.mergeRecordOverlapImpl(newRec, oldRec,
				newPos, GetTimeRangeStartIndexDescend(newTimeVals, newPos, oldTimeVals[oldRows-1])+1,
				GetTimeRangeStartIndexDescend(oldTimeVals, oldPos, newTimeVals[newPos]), oldRows,
				newTimeVals, oldTimeVals, newPos, oldPos, newRows, oldRows, limitRows, false)
		} else {
			newEnd, oldEnd = rec.mergeRecordOverlapImpl(newRec, oldRec,
				GetTimeRangeStartIndexDescend(newTimeVals, newPos, oldTimeVals[oldPos]), GetTimeRangeStartIndexDescend(newTimeVals, newPos, oldTimeVals[oldRows-1])+1,
				oldPos, oldRows,
				newTimeVals, oldTimeVals, newPos, oldPos, newRows, oldRows, limitRows, false)
		}
	} else {
		if newTimeVals[newPos] >= oldTimeVals[oldPos] {
			newEnd, oldEnd = rec.mergeRecordOverlapImpl(newRec, oldRec,
				GetTimeRangeStartIndexDescend(newTimeVals, newPos, oldTimeVals[oldPos]), newRows,
				oldPos, GetTimeRangeStartIndexDescend(oldTimeVals, oldPos, newTimeVals[newRows-1])+1,
				newTimeVals, oldTimeVals, newPos, oldPos, newRows, oldRows, limitRows, false)
		} else {
			newEnd, oldEnd = rec.mergeRecordOverlapImpl(newRec, oldRec,
				newPos, newRows, GetTimeRangeStartIndexDescend(oldTimeVals, oldPos, newTimeVals[newPos]),
				GetTimeRangeStartIndexDescend(oldTimeVals, oldPos, newTimeVals[newRows-1])+1,
				newTimeVals, oldTimeVals, newPos, oldPos, newRows, oldRows, limitRows, false)
		}
	}

	return newEnd, oldEnd
}

func (rec *Record) MergeRecord(newRec, oldRec *Record) {
	rec.MergeRecordLimitRows(newRec, oldRec, 0, 0, newRec.RowNums()+oldRec.RowNums())
}

func (rec *Record) MergeRecordDescend(newRec, oldRec *Record) {
	rec.MergeRecordLimitRowsDescend(newRec, oldRec, 0, 0, newRec.RowNums()+oldRec.RowNums())
}

func (rec *Record) MergeRecordLimitRows(newRec, oldRec *Record, newPos, oldPos, limitRows int) (int, int) {
	newTimeVals := newRec.ColVals[len(newRec.ColVals)-1].IntegerValues()
	oldTimeVals := oldRec.ColVals[len(oldRec.ColVals)-1].IntegerValues()

	var newEnd, oldEnd int
	if newTimeVals[newPos] > oldTimeVals[len(oldTimeVals)-1] {
		newEnd, oldEnd = rec.mergeRecordNonOverlap(newRec, oldRec, newPos, oldPos, len(newTimeVals), len(oldTimeVals), limitRows)
	} else if newTimeVals[len(newTimeVals)-1] < oldTimeVals[oldPos] {
		oldEnd, newEnd = rec.mergeRecordNonOverlap(oldRec, newRec, oldPos, newPos, len(oldTimeVals), len(newTimeVals), limitRows)
	} else {
		newEnd, oldEnd = rec.mergeRecordOverlap(newRec, oldRec, newTimeVals, oldTimeVals, newPos, oldPos, len(newTimeVals), len(oldTimeVals), limitRows)
	}

	return newEnd, oldEnd
}

func (rec *Record) MergeRecordLimitRowsDescend(newRec, oldRec *Record, newPos, oldPos, limitRows int) (int, int) {
	newTimeVals := newRec.ColVals[len(newRec.ColVals)-1].IntegerValues()
	oldTimeVals := oldRec.ColVals[len(oldRec.ColVals)-1].IntegerValues()

	var newEnd, oldEnd int
	if newTimeVals[newPos] < oldTimeVals[len(oldTimeVals)-1] {
		newEnd, oldEnd = rec.mergeRecordNonOverlap(newRec, oldRec, newPos, oldPos, len(newTimeVals), len(oldTimeVals), limitRows)
	} else if newTimeVals[len(newTimeVals)-1] > oldTimeVals[oldPos] {
		oldEnd, newEnd = rec.mergeRecordNonOverlap(oldRec, newRec, oldPos, newPos, len(oldTimeVals), len(newTimeVals), limitRows)
	} else {
		newEnd, oldEnd = rec.mergeRecordOverlapDescend(newRec, oldRec, newTimeVals, oldTimeVals, newPos, oldPos, len(newTimeVals), len(oldTimeVals), limitRows)
	}

	return newEnd, oldEnd
}

func (rec *Record) MergeRecordByMaxTimeOfOldRec(newRec, oldRec *Record, newPos, oldPos, limitRows int, ascending bool) (*Record, int, int) {
	newTimeVals := newRec.ColVals[len(newRec.ColVals)-1].IntegerValues()
	oldTimeVals := oldRec.ColVals[len(oldRec.ColVals)-1].IntegerValues()

	var newEnd, oldEnd int
	if ascending {
		if newTimeVals[newPos] > oldTimeVals[len(oldTimeVals)-1] {
			rec.Schema = append(rec.Schema, oldRec.Schema...)
			rec.ColVals = make([]ColVal, len(rec.Schema))
			rec.AppendRec(oldRec, oldPos, len(oldTimeVals))
			return nil, newPos, len(oldTimeVals)
		} else if newTimeVals[len(newTimeVals)-1] < oldTimeVals[oldPos] {
			oldEnd, newEnd = rec.mergeRecordNonOverlap(oldRec, newRec, oldPos, newPos, len(oldTimeVals), len(newTimeVals), limitRows)
		} else {
			newEndIndex := GetTimeRangeEndIndex(newTimeVals, newPos, oldTimeVals[len(oldTimeVals)-1])
			newTimeVals = newTimeVals[:newEndIndex+1]
			newEnd, oldEnd = rec.mergeRecordOverlap(newRec, oldRec, newTimeVals, oldTimeVals, newPos, oldPos, newEndIndex+1, len(oldTimeVals), limitRows)
		}
	} else {
		if newTimeVals[len(newTimeVals)-1] > oldTimeVals[oldPos] {
			oldEnd, newEnd = rec.mergeRecordNonOverlap(oldRec, newRec, oldPos, newPos, len(oldTimeVals), len(newTimeVals), limitRows)
		} else if newTimeVals[newPos] < oldTimeVals[len(oldTimeVals)-1] {
			rec.Schema = append(rec.Schema, oldRec.Schema...)
			rec.ColVals = make([]ColVal, len(rec.Schema))
			rec.AppendRec(oldRec, oldPos, len(oldTimeVals))
			return nil, newPos, len(oldTimeVals)
		} else {
			newEndIndex := GetTimeRangeEndIndexDescend(newTimeVals, newPos, oldTimeVals[len(oldTimeVals)-1])
			newTimeVals = newTimeVals[:newEndIndex+1]
			newEnd, oldEnd = rec.mergeRecordOverlapDescend(newRec, oldRec, newTimeVals, oldTimeVals, newPos, oldPos, newEndIndex+1, len(oldTimeVals), limitRows)
		}
	}

	return nil, newEnd, oldEnd
}

func NewRecordBuilder(schema []Field) *Record {
	return &Record{
		Schema:  schema,
		ColVals: make([]ColVal, len(schema)),
	}
}

func (rec *Record) SliceFromRecord(srcRec *Record, start, end int) {
	rec.Schema = srcRec.Schema
	// colVal mem reuse
	schemaLen := len(rec.Schema)
	if cap(rec.ColVals) < schemaLen {
		rec.ColVals = make([]ColVal, schemaLen)
	} else {
		rec.ColVals = rec.ColVals[:schemaLen]
	}
	if rec.RecMeta == nil {
		rec.RecMeta = &RecMeta{}
		rec.ColMeta = append(rec.ColMeta[:cap(rec.ColMeta)], make([]ColMeta, len(rec.Schema)-0)...)
	} else if recColMetaLen := len(rec.ColMeta); srcRec.RecMeta != nil && recColMetaLen < len(srcRec.ColMeta) {
		rec.RecMeta = &RecMeta{}
		rec.ColMeta = append(rec.ColMeta[:cap(rec.ColMeta)], make([]ColMeta, len(rec.Schema)-recColMetaLen)...)
	}
	rec.ColVals = rec.ColVals[:len(rec.Schema)]

	length := end - start
	validCount := 0
	for i := range srcRec.ColVals {
		var colValOffset int
		// support query with schema not exist
		srcCol := &srcRec.ColVals[i]
		if srcCol.Len == srcCol.NilCount {
			rec.ColVals[i].Init()
			rec.ColVals[i].PadColVal(srcRec.Schema.Field(i).Type, length)
			continue
		}

		colValOffset = srcCol.calcColumnOffset(srcRec.Schema[i].Type, start)
		_, validCount = rec.ColVals[i].sliceValAndOffset(srcCol, start, end, srcRec.Schema[i].Type, colValOffset)
		rec.ColVals[i].sliceBitMap(srcCol, start, end)
		rec.ColVals[i].Len = length
		rec.ColVals[i].NilCount = length - validCount
	}

	if srcRec.RecMeta != nil {
		srcRec.CopyColMetaTo(rec)
		for i := range srcRec.RecMeta.Times {
			if len(rec.RecMeta.Times) != 0 && len(srcRec.RecMeta.Times[i]) != 0 {
				rec.RecMeta.Times[i] = rec.RecMeta.Times[i][:0]
				rec.RecMeta.Times[i] = append(rec.RecMeta.Times[i], srcRec.RecMeta.Times[i][start:end]...)
			}
		}
	}
}

func (rec *Record) CopyColMetaTo(dst *Record) {
	if len(rec.RecMeta.Times) != 0 && len(dst.RecMeta.Times) == 0 {
		dst.RecMeta.Times = rec.RecMeta.Times
	}
	for i := range rec.ColMeta {
		dst.ColMeta[i].count = rec.ColMeta[i].count
		dst.ColMeta[i].max = rec.ColMeta[i].max
		dst.ColMeta[i].maxTime = rec.ColMeta[i].maxTime
		dst.ColMeta[i].min = rec.ColMeta[i].min
		dst.ColMeta[i].minTime = rec.ColMeta[i].minTime
		dst.ColMeta[i].sum = rec.ColMeta[i].sum
		dst.ColMeta[i].first = rec.ColMeta[i].first
		dst.ColMeta[i].firstTime = rec.ColMeta[i].firstTime
		dst.ColMeta[i].last = rec.ColMeta[i].last
		dst.ColMeta[i].lastTime = rec.ColMeta[i].lastTime
	}
}

func (rec *Record) Times() []int64 {
	if len(rec.ColVals) == 0 {
		return nil
	}
	cv := rec.ColVals[len(rec.ColVals)-1]
	return cv.IntegerValues()
}

func subBitmapBytes(bitmap []byte, bitMapOffset int, length int) ([]byte, int) {
	if ((bitMapOffset + length) & 0x7) != 0 {
		return bitmap[bitMapOffset>>3 : ((bitMapOffset+length)>>3 + 1)], bitMapOffset & 0x7
	}

	return bitmap[bitMapOffset>>3 : (bitMapOffset+length)>>3], bitMapOffset & 0x7
}

func valueIndexRange(bitMap []byte, bitOffset int, bmStart, bmEnd int) (valStart, valEnd int) {
	var start, end int
	for i := 0; i < bmEnd; i++ {
		if bitMap[(bitOffset+i)>>3]&BitMask[(bitOffset+i)&0x07] != 0 {
			if i < bmStart {
				start++
			}
			end++
		}
	}
	return start, end
}

func (rec *Record) String() string {
	var sb strings.Builder

	for i, f := range rec.Schema {
		var line string
		switch f.Type {
		case influx.Field_Type_Float:
			line = fmt.Sprintf("field(%v):%#v\n", f.Name, rec.Column(i).FloatValues())
		case influx.Field_Type_String, influx.Field_Type_Tag:
			line = fmt.Sprintf("field(%v):%#v\n", f.Name, rec.Column(i).StringValues(nil))
		case influx.Field_Type_Boolean:
			line = fmt.Sprintf("field(%v):%#v\n", f.Name, rec.Column(i).BooleanValues())
		case influx.Field_Type_Int:
			line = fmt.Sprintf("field(%v):%#v\n", f.Name, rec.Column(i).IntegerValues())
		}

		sb.WriteString(line)
	}

	return sb.String()
}

func (rec *Record) ReserveColumnRows(rows int) {
	for i := range rec.Schema {
		schema := &rec.Schema[i]
		col := &rec.ColVals[i]
		l := len(col.Val)
		switch schema.Type {
		case influx.Field_Type_Float, influx.Field_Type_Int:
			size := rows * 8
			if cap(col.Val) < size {
				newCol := make([]byte, size)
				copy(newCol, col.Val[:l])
				col.Val = newCol
			}
		case influx.Field_Type_Boolean:
			if cap(col.Val) < rows {
				newCol := make([]byte, rows)
				copy(newCol, col.Val[:l])
				col.Val = newCol
			}
		case influx.Field_Type_String:
			size := rows * 16
			if cap(col.Val) < size {
				newCol := make([]byte, size)
				copy(newCol, col.Val[:l])
				col.Val = newCol
			}

			offLen := len(col.Offset)
			if cap(col.Offset) < rows {
				newOff := make([]uint32, rows)
				copy(newOff, col.Offset[:offLen])
				col.Offset = newOff
				col.Offset = col.Offset[:offLen]
			}

		default:
			panic(fmt.Sprintf("unknown column data type %v::%v", schema.Name, schema.Type))
		}

		bitLen := len(col.Bitmap)
		bitBytes := (rows + 7) / 8
		if cap(col.Bitmap) < bitBytes {
			newBit := make([]byte, bitBytes)
			copy(newBit, col.Bitmap[:bitLen])
			col.Bitmap = newBit
		}
		col.Bitmap = col.Bitmap[:bitLen]
		col.Val = col.Val[:l]
	}
}

func (rec *Record) Reset() {
	rec.Schema = rec.Schema[:0]
	rec.ColVals = rec.ColVals[:0]
}

func (rec *Record) ResetDeep() {
	for i := range rec.ColVals {
		rec.ColVals[i].Init()
	}
	rec.Schema = rec.Schema[:0]
	rec.ColVals = rec.ColVals[:0]
	if rec.RecMeta != nil {
		for i := range rec.ColMeta {
			rec.ColMeta[i].Init()
		}
		rec.tags = rec.tags[:0]
		rec.IntervalIndex = rec.IntervalIndex[:0]
		rec.tagIndex = rec.tagIndex[:0]
		rec.ColMeta = rec.ColMeta[:0]
		rec.RecMeta.Times = rec.RecMeta.Times[:0]
	}
}

func (rec *Record) ResetForReuse() {
	for i := range rec.ColVals {
		rec.ColVals[i].Init()
	}
	if rec.RecMeta != nil {
		for i := range rec.ColMeta {
			rec.ColMeta[i].Init()
		}
		rec.tags = rec.tags[:0]
		rec.tagIndex = rec.tagIndex[:0]
		rec.IntervalIndex = rec.IntervalIndex[:0]
		for i := range rec.RecMeta.Times {
			rec.RecMeta.Times[i] = rec.RecMeta.Times[i][:0]
		}
	}
}

func (rec *Record) Reuse() {
	for i := range rec.ColVals {
		rec.ColVals[i].Init()
	}
}

func (rec *Record) ResetWithSchema(schema Schemas) {
	rec.Reset()
	rec.Schema = schema
	rec.ReserveColVal(len(rec.Schema))
}

func (rec *Record) addColumn(f *Field) {
	newField := Field{Name: f.Name, Type: f.Type}
	newCol := &ColVal{}
	for i := 0; i < rec.RowNums(); i++ {
		switch f.Type {
		case influx.Field_Type_Int:
			newCol.AppendIntegerNull()
		case influx.Field_Type_Float:
			newCol.AppendFloatNull()
		case influx.Field_Type_Boolean:
			newCol.AppendBooleanNull()
		case influx.Field_Type_String:
			newCol.AppendStringNull()
		}
	}

	rec.Schema = append(rec.Schema, newField)
	rec.ColVals = append(rec.ColVals, *newCol)
}

func (rec *Record) PadRecord(other *Record) {
	needSort := false
	for i := range other.Schema[:len(other.Schema)-1] {
		f := &(other.Schema[i])
		idx := rec.FieldIndexs(f.Name)
		if idx < 0 {
			needSort = true
			rec.addColumn(f)
		}
	}

	if needSort {
		sort.Sort(rec)
	}
}

// Merge only for level compaction use
func (rec *Record) Merge(newRec *Record) {
	rec.PadRecord(newRec)
	newRecRows := newRec.RowNums()
	oldColumnN, newColumnN := rec.ColNums(), newRec.ColNums()
	oldIdx, newIdx := 0, 0
	for oldIdx < oldColumnN-1 && newIdx < newColumnN-1 {
		col := &(rec.ColVals[oldIdx])
		newCol := &(newRec.ColVals[newIdx])

		if rec.Schema[oldIdx].Name == newRec.Schema[newIdx].Name {
			col.AppendColVal(newCol, rec.Schema[oldIdx].Type, 0, newRecRows)
			oldIdx++
			newIdx++
		} else if rec.Schema[oldIdx].Name < newRec.Schema[newIdx].Name {
			col.PadColVal(rec.Schema[oldIdx].Type, newRecRows)
			oldIdx++
		}
	}

	for oldIdx < oldColumnN-1 {
		col := &(rec.ColVals[oldIdx])
		col.PadColVal(rec.Schema[oldIdx].Type, newRecRows)
		oldIdx++
	}

	col := &(rec.ColVals[oldColumnN-1])
	newCol := &(newRec.ColVals[newColumnN-1])
	col.AppendColVal(newCol, rec.Schema[oldColumnN-1].Type, 0, newRecRows)
}

func (rec *Record) IsNilRow(row int) bool {
	// exclude time column
	colNum := rec.ColNums() - 1
	for j := 0; j < colNum; j++ {
		if !rec.ColVals[j].IsNil(row) {
			return false
		}
	}
	return true
}

func (rec *Record) KickNilRow() *Record {
	// fast path, no need to kick
	colNum := rec.ColNums() - 1
	for i := 0; i < colNum; i++ {
		if rec.ColVals[i].Len == 0 {
			continue
		}
		if rec.ColVals[i].NilCount == 0 {
			return rec
		}
	}

	// slow path, try to kick
	rowNum := rec.RowNums()
	isFirst := true
	var newRec *Record
	for rowIdx := 0; rowIdx < rowNum; {
		startRow := rowIdx
		endRow := rowIdx
		for endRow < rowNum && !rec.IsNilRow(endRow) {
			endRow++
		}
		if endRow != startRow {
			// all rows are not nil row
			if endRow-startRow == rowNum {
				return rec
			}
			if isFirst {
				newRec = NewRecordBuilder(rec.Schema)
				isFirst = false
			}
			newRec.AppendRec(rec, startRow, endRow)
			rowIdx = endRow
		} else {
			rowIdx++
		}
	}

	if newRec == nil {
		newRec = NewRecordBuilder(rec.Schema)
	}

	return newRec
}

func (rec *Record) TryPadColumn() {
	rows := rec.RowNums()
	for i := range rec.ColVals {
		if rec.ColVals[i].Len != rows {
			rec.ColVals[i].PadColVal(rec.Schema[i].Type, rows-rec.ColVals[i].Len)
		}
	}
}

func (rec *Record) Size() int {
	size := 0
	for i := range rec.Schema {
		size += len(rec.Schema[i].Name)
		size += len(rec.ColVals[i].Val)
		if len(rec.ColVals[i].Offset) > 0 {
			size += len(rec.ColVals[i].Offset) * 4
		}
	}

	return size
}

func (rec *Record) AddTagIndexAndKey(key *[]byte, i int) {
	rec.tagIndex = append(rec.tagIndex, i)
	rec.tags = append(rec.tags, key)
}

func (rec *Record) GetTagIndexAndKey() ([]*[]byte, []int) {
	return rec.tags, rec.tagIndex
}

func (rec *Record) IntervalFirstTime() int64 {
	return rec.ColVals[len(rec.ColVals)-1].IntegerValues()[0]
}

func (rec *Record) IntervalLastTime() int64 {
	return rec.ColVals[len(rec.ColVals)-1].IntegerValues()[rec.RowNums()-1]
}

func (rec *Record) AppendIntervalEmptyRows(start, step, num int64, initRecMeta bool) {
	for i := int64(0); i < num; i++ {
		rec.AppendIntervalEmptyRow(start+step*i, initRecMeta)
	}
}

func (rec *Record) AppendIntervalEmptyRow(rowTime int64, initRecMeta bool) {
	for i := 0; i < rec.Len()-1; i++ {
		switch rec.Schema[i].Type {
		case influx.Field_Type_Float:
			rec.ColVals[i].AppendFloatNullReserve()
		case influx.Field_Type_String, influx.Field_Type_Tag:
			rec.ColVals[i].AppendStringNull()
		case influx.Field_Type_Int:
			rec.ColVals[i].AppendIntegerNullReserve()
		case influx.Field_Type_Boolean:
			rec.ColVals[i].AppendBooleanNullReserve()
		default:
			panic("unsupported data type")
		}
	}
	rec.ColVals[len(rec.ColVals)-1].AppendInteger(rowTime)
	if initRecMeta {
		for i := 0; i < len(rec.RecMeta.Times)-1; i++ {
			rec.RecMeta.Times[i] = append(rec.RecMeta.Times[i], 0)
		}
	}
}

func (rec *Record) AppendRecRow2IntervalRec(re *Record, row int) {
	for i := 0; i < len(re.Schema)-1; i++ {
		intervalRecAppendFunctions[re.Schema[i].Type](re, rec, i, row)
	}
	time, _ := re.ColVals[len(re.Schema)-1].IntegerValue(row)
	rec.ColVals[len(rec.Schema)-1].AppendInteger(time)
	appendRecMeta2iRec(re, rec, len(re.Schema)-1, row)
}

func (rec *Record) BuildEmptyIntervalRec(min, max, interval int64, initRecMeta, hasInterval, ascending bool) {
	if !hasInterval {
		rec.AppendIntervalEmptyRows(0, interval, 1, initRecMeta)
		return
	}
	num := (max - min) / interval
	if ascending {
		rec.AppendIntervalEmptyRows(min, interval, num, initRecMeta)
	} else {
		rec.AppendIntervalEmptyRows(max-interval, -interval, num, initRecMeta)
	}
}

func (rec *Record) TransIntervalRec2Rec(re *Record, start, end int) {
	for i := start; i < end; i++ {
		if rec.IsIntervalRecRowNull(i) {
			continue
		}
		for j := range rec.Schema {
			recTransAppendFunctions[rec.Schema[j].Type](re, rec, j, i)
		}
	}
}

func (rec *Record) IsIntervalRecRowNull(row int) bool {
	for i := 0; i < rec.Len()-1; i++ {
		if !rec.ColVals[i].IsNil(row) {
			return false
		}
	}
	return true
}

func (rec *Record) UpdateIntervalRecRow(re *Record, recRow, row int) {
	for i := range re.Schema {
		intervalRecUpdateFunctions[re.Schema[i].Type](re, rec, i, row, recRow)
	}
}

func (rec *Record) ResizeBySchema(schema Schemas, initColMeta bool) {
	capCol := cap(rec.ColVals)
	if capCol < len(schema) {
		padLen := len(schema) - capCol
		rec.ColVals = append(rec.ColVals, make([]ColVal, padLen)...)
		for i := cap(rec.ColVals) - padLen; i < rec.Len(); i++ {
			rec.ColVals[i] = ColVal{}
		}
		if initColMeta {
			rec.RecMeta.Times = append(rec.RecMeta.Times, make([][]int64, padLen)...)
			for i := cap(rec.ColVals) - padLen; i < rec.Len(); i++ {
				rec.RecMeta.Times[i] = make([]int64, 0, len(rec.RecMeta.Times[0]))
			}
		}
	}
	rec.ColVals = rec.ColVals[:len(schema)]
	if initColMeta {
		rec.RecMeta.Times = rec.RecMeta.Times[:len(schema)]
	}
}

func ReverseBitMap(bitmap []byte, bitmapOffset uint32, count int) []byte {
	if len(bitmap) == 0 {
		return bitmap
	}
	left, right := 0+int(bitmapOffset), count-1+int(bitmapOffset)
	for left < right {
		bitLeft := (bitmap[left>>3] & BitMask[left&0x07]) == 0
		bitRight := (bitmap[right>>3] & BitMask[right&0x07]) == 0

		if bitLeft {
			bitmap[right>>3] &= FlippedBitMask[right&0x07]
		} else {
			bitmap[right>>3] |= BitMask[right&0x07]
		}

		if bitRight {
			bitmap[left>>3] &= FlippedBitMask[left&0x07]
		} else {
			bitmap[left>>3] |= BitMask[left&0x07]
		}

		left++
		right--
	}
	return bitmap
}

type RecordPool struct {
	name  RecordType
	cache chan *Record
	pool  sync.Pool
	inUse func(i int64)
	get   func(i int64)
	reUse func(i int64)
	abort func(i int64)
}

type RecordType uint8

const (
	IntervalRecordPool RecordType = iota
	FileCursorPool
	AggPool
	TsmMergePool
	TsspSequencePool
	SequenceAggPool
	SeriesPool
	UnknownPool
)

func NewRecordPool(recordType RecordType) *RecordPool {
	n := cpu.GetCpuNum() * 2
	if n < 4 {
		n = 4
	}
	if n > 256 {
		n = 256
	}

	inUse := func(i int64) {}
	get := func(i int64) {}
	reuse := func(i int64) {}
	abort := func(i int64) {}
	switch recordType {
	case IntervalRecordPool:
		inUse = statistics.NewRecordStatistics().AddIntervalRecordPoolInUse
		get = statistics.NewRecordStatistics().AddIntervalRecordPoolGet
		reuse = statistics.NewRecordStatistics().AddIntervalRecordPoolGetReUse
		abort = statistics.NewRecordStatistics().AddIntervalRecordPoolAbort
	case FileCursorPool:
		inUse = statistics.NewRecordStatistics().AddFileCursorPoolInUse
		get = statistics.NewRecordStatistics().AddFileCursorPoolGet
		reuse = statistics.NewRecordStatistics().AddFileCursorPoolGetReUse
		abort = statistics.NewRecordStatistics().AddFileCursorPoolAbort
	case AggPool:
		inUse = statistics.NewRecordStatistics().AddAggPoolInUse
		get = statistics.NewRecordStatistics().AddAggPoolGet
		reuse = statistics.NewRecordStatistics().AddAggPoolGetReUse
		abort = statistics.NewRecordStatistics().AddAggPoolAbort
	case TsmMergePool:
		inUse = statistics.NewRecordStatistics().AddTsmMergePoolInUse
		get = statistics.NewRecordStatistics().AddTsmMergePoolGet
		reuse = statistics.NewRecordStatistics().AddTsmMergePoolGetReUse
		abort = statistics.NewRecordStatistics().AddTsmMergePoolAbort
	case TsspSequencePool:
		inUse = statistics.NewRecordStatistics().AddTsspSequencePoolInUse
		get = statistics.NewRecordStatistics().AddTsspSequencePoolGet
		reuse = statistics.NewRecordStatistics().AddTsspSequencePoolGetReUse
		abort = statistics.NewRecordStatistics().AddTsspSequencePoolAbort
	case SequenceAggPool:
		inUse = statistics.NewRecordStatistics().AddSequenceAggPoolInUse
		get = statistics.NewRecordStatistics().AddSequenceAggPoolGet
		reuse = statistics.NewRecordStatistics().AddSequenceAggPoolGetReUse
		abort = statistics.NewRecordStatistics().AddSequenceAggPoolAbort
	case SeriesPool:
		inUse = statistics.NewRecordStatistics().AddSeriesPoolInUse
		get = statistics.NewRecordStatistics().AddSeriesPoolGet
		reuse = statistics.NewRecordStatistics().AddSeriesPoolGetReUse
		abort = statistics.NewRecordStatistics().AddSeriesPoolAbort
	}

	return &RecordPool{
		cache: make(chan *Record, n),
		name:  recordType,
		inUse: inUse,
		get:   get,
		reUse: reuse,
		abort: abort,
	}
}

func (p *RecordPool) Get() *Record {
	p.inUse(1)
	p.get(1)
	select {
	case rec := <-p.cache:
		p.reUse(1)
		return rec
	default:
		v := p.pool.Get()
		if v != nil {
			rec, ok := v.(*Record)
			if !ok {
				return &Record{}
			}
			p.reUse(1)
			return rec
		}
		rec := &Record{}
		return rec
	}
}

func (p *RecordPool) Put(rec *Record) {
	p.inUse(-1)
	if recLen := rec.Len(); (recLen > 0 && rec.RowNums() > RecMaxRowNumForRuse) || cap(rec.Schema) > RecMaxLenForRuse {
		p.abort(1)
		return
	}
	rec.ResetDeep()
	select {
	case p.cache <- rec:
	default:
		p.pool.Put(rec)
	}
}

type CircularRecordPool struct {
	index       int
	recordNum   int
	pool        *RecordPool
	records     []*Record
	initColMeta bool
}

func NewCircularRecordPool(recordPool *RecordPool, recordNum int, schema Schemas, initColMeta bool) *CircularRecordPool {
	statistics.NewRecordStatistics().AddCircularRecordPool(1)
	rp := &CircularRecordPool{
		index:       0,
		recordNum:   recordNum,
		pool:        recordPool,
		initColMeta: initColMeta,
	}

	schemaLen := schema.Len()
	for i := 0; i < recordNum; i++ {
		record := recordPool.Get()
		recSchemaLen := len(record.Schema)
		if schemaCap := cap(record.Schema); schemaCap < schemaLen {
			record.Schema = append(record.Schema[:recSchemaLen], make([]Field, schemaLen-recSchemaLen)...)
		}
		record.Schema = record.Schema[:schemaLen]

		if colValCap := cap(record.ColVals); colValCap < schemaLen {
			record.ColVals = append(record.ColVals[:recSchemaLen], make([]ColVal, schemaLen-recSchemaLen)...)
		}
		record.ColVals = record.ColVals[:schemaLen]

		copy(record.Schema, schema)
		if initColMeta {
			if record.RecMeta == nil {
				record.RecMeta = &RecMeta{
					Times: make([][]int64, schemaLen),
				}
			}
			if timeCap, timeLen := cap(record.RecMeta.Times), len(record.RecMeta.Times); timeCap < schemaLen {
				record.RecMeta.Times = append(record.RecMeta.Times[:timeLen], make([][]int64, schemaLen-timeLen)...)
			}
			record.RecMeta.Times = record.RecMeta.Times[:schemaLen]
		}
		rp.records = append(rp.records, record)
	}
	return rp
}

func (p *CircularRecordPool) GetIndex() int {
	return p.index
}

func (p *CircularRecordPool) Get() *Record {
	r := p.records[p.index]
	r.ResetForReuse()
	if !p.initColMeta {
		r.RecMeta = nil
	}
	p.index = (p.index + 1) % p.recordNum
	return r
}

func (p *CircularRecordPool) GetBySchema(s Schemas) *Record {
	r := p.records[p.index]
	r.Schema = s
	r.ResetForReuse()
	if !p.initColMeta {
		r.RecMeta = nil
	}
	p.index = (p.index + 1) % p.recordNum
	return r
}

func (p *CircularRecordPool) Put() {
	statistics.NewRecordStatistics().AddCircularRecordPool(-1)
	for i := 0; i < p.recordNum; i++ {
		p.pool.Put(p.records[i])
	}
	p.pool = nil
	p.records = nil
	p.recordNum = 0
}

func (p *CircularRecordPool) PutRecordInCircularPool() {
	p.index = (p.index - 1 + p.recordNum) % p.recordNum
}

func (rec *Record) Split(dst []Record, maxRows int) []Record {
	if rec.RowNums() <= maxRows {
		dst = append(dst[:0], *rec)
		return dst
	}

	rows := rec.RowNums()
	segs := (rows + maxRows - 1) / maxRows
	if cap(dst) < segs {
		delta := segs - cap(dst)
		dst = dst[:cap(dst)]
		dst = append(dst, make([]Record, delta)...)
	}
	dst = dst[:segs]

	if segs == 1 {
		dst[0] = *rec
		return dst
	}

	for i := range dst {
		dst[i].Schema = append(dst[i].Schema[:0], rec.Schema...)
		dst[i].ColVals = resize(dst[i].ColVals, rec.Schema.Len())
	}

	for i := range rec.Schema {
		col := rec.Column(i)
		dstCol := col.Split(nil, maxRows, rec.Schema[i].Type)
		for j := range dstCol {
			dst[j].ColVals[i] = dstCol[j]
		}
	}

	return dst
}

func stringAppendFunction(rec, iRec *Record, index, row int) {
	v, isNil := rec.ColVals[index].StringValueUnsafe(row)
	if isNil {
		iRec.ColVals[index].AppendStringNull()
	} else {
		iRec.ColVals[index].AppendString(v)
	}
	appendRecMeta2iRec(rec, iRec, index, row)
}

func integerAppendFunction(rec, iRec *Record, index, row int) {
	v, isNil := rec.ColVals[index].IntegerValue(row)
	if isNil {
		iRec.ColVals[index].AppendIntegerNullReserve()
	} else {
		iRec.ColVals[index].AppendInteger(v)
	}
	appendRecMeta2iRec(rec, iRec, index, row)
}
func floatAppendFunction(rec, iRec *Record, index, row int) {
	v, isNil := rec.ColVals[index].FloatValue(row)
	if isNil {
		iRec.ColVals[index].AppendFloatNullReserve()
	} else {
		iRec.ColVals[index].AppendFloat(v)
	}
	appendRecMeta2iRec(rec, iRec, index, row)
}
func booleanAppendFunction(rec, iRec *Record, index, row int) {
	v, isNil := rec.ColVals[index].BooleanValue(row)
	if isNil {
		iRec.ColVals[index].AppendBooleanNullReserve()
	} else {
		iRec.ColVals[index].AppendBoolean(v)
	}
	appendRecMeta2iRec(rec, iRec, index, row)
}

func stringUpdateFunction(rec, iRec *Record, index, row, recRow int) {
	v, isNil := rec.ColVals[index].StringValueUnsafe(recRow)
	iRec.ColVals[index].UpdateStringValue(v, isNil, row)
	updateRecMeta(rec, iRec, index, row, recRow)
}

func integerUpdateFunction(rec, iRec *Record, index, row, recRow int) {
	v, isNil := rec.ColVals[index].IntegerValue(recRow)
	iRec.ColVals[index].UpdateIntegerValue(v, isNil, row)
	updateRecMeta(rec, iRec, index, row, recRow)
}

func floatUpdateFunction(rec, iRec *Record, index, row, recRow int) {
	v, isNil := rec.ColVals[index].FloatValue(recRow)
	iRec.ColVals[index].UpdateFloatValue(v, isNil, row)
	updateRecMeta(rec, iRec, index, row, recRow)
}

func booleanUpdateFunction(rec, iRec *Record, index, row, recRow int) {
	v, isNil := rec.ColVals[index].BooleanValue(recRow)
	iRec.ColVals[index].UpdateBooleanValue(v, isNil, row)
	updateRecMeta(rec, iRec, index, row, recRow)
}

func recStringAppendFunction(rec, iRec *Record, index, row int) {
	v, isNil := iRec.ColVals[index].StringValueUnsafe(row)
	if isNil {
		rec.ColVals[index].AppendStringNull()
	} else {
		rec.ColVals[index].AppendString(v)
	}
	appendRecMeta2Rec(rec, iRec, index, row)
}

func recIntegerAppendFunction(rec, iRec *Record, index, row int) {
	v, isNil := iRec.ColVals[index].IntegerValueWithNullReserve(row)
	if isNil {
		rec.ColVals[index].AppendIntegerNull()
	} else {
		rec.ColVals[index].AppendInteger(v)
	}
	appendRecMeta2Rec(rec, iRec, index, row)
}

func recFloatAppendFunction(rec, iRec *Record, index, row int) {
	v, isNil := iRec.ColVals[index].FloatValueWithNullReserve(row)
	if isNil {
		rec.ColVals[index].AppendFloatNull()
	} else {
		rec.ColVals[index].AppendFloat(v)
	}
	appendRecMeta2Rec(rec, iRec, index, row)
}

func recBooleanAppendFunction(rec, iRec *Record, index, row int) {
	v, isNil := iRec.ColVals[index].BooleanValueWithNullReserve(row)
	if isNil {
		rec.ColVals[index].AppendBooleanNull()
	} else {
		rec.ColVals[index].AppendBoolean(v)
	}
	appendRecMeta2Rec(rec, iRec, index, row)
}

func appendRecMeta2iRec(rec, iRec *Record, index, row int) {
	if rec.RecMeta != nil && len(rec.RecMeta.Times) != 0 && len(rec.RecMeta.Times[index]) != 0 {
		iRec.RecMeta.Times[index] = append(iRec.RecMeta.Times[index], rec.RecMeta.Times[index][row])
	}
}

func updateRecMeta(rec, iRec *Record, index, row, recRow int) {
	if rec.RecMeta != nil && len(rec.RecMeta.Times) != 0 && index < len(rec.RecMeta.Times)-1 &&
		len(rec.RecMeta.Times[index]) != 0 {
		iRec.RecMeta.Times[index][row] = rec.RecMeta.Times[index][recRow]
	}
}

func appendRecMeta2Rec(rec, iRec *Record, index, row int) {
	if rec.RecMeta != nil && len(iRec.RecMeta.Times) != 0 && len(iRec.RecMeta.Times[index]) != 0 {
		rec.RecMeta.Times[index] = append(rec.RecMeta.Times[index], iRec.RecMeta.Times[index][row])
	}
}
