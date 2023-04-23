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

package immutable

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/influxdata/influxdb/pkg/bloom"
	"github.com/openGemini/openGemini/engine/immutable/encoding"
	"github.com/openGemini/openGemini/lib/fileops"
	"github.com/openGemini/openGemini/lib/interruptsignal"
	"github.com/openGemini/openGemini/lib/rand"
	"github.com/openGemini/openGemini/lib/record"
	"github.com/openGemini/openGemini/lib/util"
	"github.com/openGemini/openGemini/open_src/vm/protoparser/influx"
	"github.com/pingcap/failpoint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testDir = "/tmp/data1"
)

func genMemTableData(id uint64, idCount int, rows int, idr *MinMax, tr *MinMax) ([]uint64, map[uint64]*record.Record) {
	tm := time.Now().Truncate(time.Minute).UnixNano()

	idr.min = id
	tr.min = uint64(tm)

	schema := []record.Field{
		{Name: "field1_int64", Type: influx.Field_Type_Int},
		{Name: "field2_float", Type: influx.Field_Type_Float},
		{Name: "field3_string", Type: influx.Field_Type_String},
		{Name: "field4_bool", Type: influx.Field_Type_Boolean},
		{Name: "time", Type: influx.Field_Type_Int},
	}

	genRecFn := func() *record.Record {
		b := record.NewRecordBuilder(schema)

		f1 := rand.Int63n(10)
		f2 := 1.2 * float64(f1)
		f4 := true

		f1Builder := b.Column(0) // int64
		f2Builder := b.Column(1) // float
		f3Builder := b.Column(2) // string
		f4Builder := b.Column(3) // bool
		tmBuilder := b.Column(4) // timestamp
		for i := 1; i <= rows; i++ {
			if i%21 == 0 {
				f1Builder.AppendIntegerNull()
			} else {
				f1Builder.AppendInteger(f1)
			}

			f2Builder.AppendFloat(f2)

			if i%25 == 0 {
				f3Builder.AppendStringNull()
			} else {
				f3 := fmt.Sprintf("test_%d", f1)
				f3Builder.AppendString(f3)
			}

			if i%30 == 0 {
				f4Builder.AppendBooleanNull()
			} else {
				f4Builder.AppendBoolean(f4)
			}

			tmBuilder.AppendInteger(tm)
			f4 = !f4
			tm += time.Millisecond.Milliseconds()
			f1++
			f2 += 1.1
		}

		return b
	}

	ids := make([]uint64, 0, idCount)
	data := make(map[uint64]*record.Record, idCount)
	for i := 0; i < idCount; i++ {
		rec := genRecFn()
		data[id] = rec
		ids = append(ids, id)
		id++
	}

	idr.max = id - 1
	tr.max = uint64(tm - time.Millisecond.Milliseconds())

	return ids, data
}

func TestTableStoreOpen(t *testing.T) {
	sig := interruptsignal.NewInterruptSignal()
	defer func() {
		sig.Close()
		fileops.RemoveAll(testDir)
	}()
	_ = fileops.RemoveAll(testDir)
	conf := NewConfig()
	tier := uint64(util.Hot)
	lockPath := ""
	store := NewTableStore(testDir, &lockPath, &tier, false, conf)

	write := func(ids []uint64, data map[uint64]*record.Record, msb *MsBuilder) {
		for _, id := range ids {
			rec := data[id]
			err := msb.WriteData(id, rec)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	fileSeq := uint64(1)
	msts := []string{"mst", "mst1", "cpu"}
	var idMinMax, tmMinMax MinMax
	for _, mst := range msts {
		ids, data := genMemTableData(1, 10, 100, &idMinMax, &tmMinMax)
		fileName := NewTSSPFileName(fileSeq, 0, 0, 0, true, &lockPath)
		msb := NewMsBuilder(testDir, mst, &lockPath, conf, 10, fileName, 0, store.Sequencer(), 2)
		write(ids, data, msb)
		fileSeq++
		store.AddTable(msb, true, false)
	}

	sid := uint64(10)
	for i := 0; i < 3; i++ {
		ids, data := genMemTableData(sid, 10, 100, &idMinMax, &tmMinMax)
		isOrder := !(i == 2)
		fileName := NewTSSPFileName(fileSeq, 0, 0, 0, isOrder, &lockPath)
		msb := NewMsBuilder(testDir, "mst", &lockPath, conf, 10, fileName, 0, store.Sequencer(), 2)
		write(ids, data, msb)
		sid += 5
		store.AddTable(msb, isOrder, false)
		fileSeq++
	}

	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	tier1 := uint64(util.Hot)
	store = NewTableStore(testDir, &lockPath, &tier1, false, conf)
	if _, err := store.Open(); err != nil {
		t.Fatal(err)
	}
}

func TestMemoryRead(t *testing.T) {
	sig := interruptsignal.NewInterruptSignal()
	defer func() {
		sig.Close()
		fileops.RemoveAll(testDir)
	}()
	_ = fileops.RemoveAll(testDir)
	conf := NewConfig()
	conf.cacheDataBlock = true
	conf.cacheMetaData = true
	tier := uint64(util.Hot)
	lockPath := ""
	store := NewTableStore(testDir, &lockPath, &tier, false, conf)

	write := func(ids []uint64, data map[uint64]*record.Record, msb *MsBuilder) {
		for _, id := range ids {
			rec := data[id]
			err := msb.WriteData(id, rec)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	fileSeq := uint64(1)
	var idMinMax, tmMinMax MinMax
	ids, data := genMemTableData(1, 10, 100, &idMinMax, &tmMinMax)
	fileName := NewTSSPFileName(fileSeq, 0, 0, 0, true, &lockPath)
	msb := NewMsBuilder(testDir, "mst", &lockPath, conf, 10, fileName, 0, store.Sequencer(), 2)
	write(ids, data, msb)
	fileSeq++
	store.AddTable(msb, true, false)

	fs := store.tableFiles("mst", true)
	if fs == nil {
		t.Fatal("get mst files fail")
	}

	f := store.File("mst", fs.Files()[0].Path(), true)
	if f == nil {
		t.Fatal("get file fail")
	}
	midx, _ := f.MetaIndexAt(0)
	if midx == nil {
		t.Fatalf("meta index not find")
	}
	decs := NewReadContext(true)
	cms, err := f.ReadChunkMetaData(0, midx, nil)
	if err != nil {
		t.Fatal(err)
	}
	rec := record.NewRecordBuilder(schema)
	fr := f.(*tsspFile).reader.(*tsspFileReader)
	fr.Unref()

	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			fr.Ref()
			_, err = f.ReadAt(&cms[0], 0, rec, decs)
			if err != nil {
				t.Error(err)
			}
			fr.Unref()
			defer wg.Done()
		}()
	}
	wg.Wait()
	if fr.ref != 0 {
		t.Fatal("ref error")
	}

	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestLazyInitError(t *testing.T) {
	sig := interruptsignal.NewInterruptSignal()
	defer func() {
		sig.Close()
		fileops.RemoveAll(testDir)
	}()
	_ = fileops.RemoveAll(testDir)
	conf := NewConfig()
	tier := uint64(util.Hot)
	lockPath := ""
	store := NewTableStore(testDir, &lockPath, &tier, false, conf)

	write := func(ids []uint64, data map[uint64]*record.Record, msb *MsBuilder) {
		for _, id := range ids {
			rec := data[id]
			err := msb.WriteData(id, rec)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	fileSeq := uint64(1)
	var idMinMax, tmMinMax MinMax
	ids, data := genMemTableData(1, 10, 100, &idMinMax, &tmMinMax)
	fileName := NewTSSPFileName(fileSeq, 0, 0, 0, true, &lockPath)
	msb := NewMsBuilder(testDir, "mst", &lockPath, conf, 10, fileName, 0, store.Sequencer(), 2)
	write(ids, data, msb)
	fileSeq++
	store.AddTable(msb, true, false)

	fp := struct {
		failPath string
		inTerms  string
		expect   func(err error) error
	}{
		failPath: "github.com/openGemini/openGemini/engine/immutable/lazyInit-error",
		inTerms:  fmt.Sprintf(`return("%s")`, "lazyInit error"),
		expect: func(err error) error {
			if err != nil && err.Error() == fmt.Sprintf("%s", "lazyInit error") {
				return nil
			}
			return fmt.Errorf("unexpected error:%s", err)
		},
	}

	fs := store.tableFiles("mst", true)
	if fs == nil {
		t.Fatal("get mst files fail")
	}

	f := store.File("mst", fs.Files()[0].Path(), true)
	if f == nil {
		t.Fatal("get file fail")
	}

	require.NoError(t, failpoint.Enable(fp.failPath, fp.inTerms))
	midx, err := f.MetaIndexAt(0)
	if err = fp.expect(err); err != nil {
		t.Fatal(err)
	}
	require.NoError(t, failpoint.Disable(fp.failPath))

	midx, err = f.MetaIndexAt(0)
	if err != nil {
		t.Fatal(err)
	}
	decs := NewReadContext(true)
	require.NoError(t, failpoint.Enable(fp.failPath, fp.inTerms))
	cms, err := f.ReadChunkMetaData(0, midx, nil)
	if err = fp.expect(err); err != nil {
		t.Fatal(err)
	}
	require.NoError(t, failpoint.Disable(fp.failPath))
	cms, _ = f.ReadChunkMetaData(0, midx, nil)
	rec := record.NewRecordBuilder(schema)
	fr := f.(*tsspFile).reader.(*tsspFileReader)
	fr.Unref()

	require.NoError(t, failpoint.Enable(fp.failPath, fp.inTerms))
	fr.Ref()
	_, err = f.ReadAt(&cms[0], 0, rec, decs)
	if err = fp.expect(err); err != nil {
		t.Fatal(err)
	}
	fr.Unref()
	if fr.ref != 0 {
		t.Fatal("ref error")
	}
	require.NoError(t, failpoint.Disable(fp.failPath))

	require.NoError(t, failpoint.Enable(fp.failPath, fp.inTerms))
	fr.Ref()
	_, err = f.ReadData(0, 1, nil)
	if err = fp.expect(err); err != nil {
		t.Fatal(err)
	}
	fr.Unref()
	require.NoError(t, failpoint.Disable(fp.failPath))
	if fr.ref != 0 {
		t.Fatal("ref error")
	}

	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestMemoryReadReload(t *testing.T) {
	sig := interruptsignal.NewInterruptSignal()
	defer func() {
		sig.Close()
		fileops.RemoveAll(testDir)
	}()
	_ = fileops.RemoveAll(testDir)
	conf := NewConfig()
	conf.cacheDataBlock = true
	conf.cacheMetaData = true
	tier := uint64(util.Hot)
	lockPath := ""
	store := NewTableStore(testDir, &lockPath, &tier, false, conf)

	write := func(ids []uint64, data map[uint64]*record.Record, msb *MsBuilder) {
		for _, id := range ids {
			rec := data[id]
			err := msb.WriteData(id, rec)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	fileSeq := uint64(1)
	var idMinMax, tmMinMax MinMax
	ids, data := genMemTableData(1, 10, 100, &idMinMax, &tmMinMax)
	fileName := NewTSSPFileName(fileSeq, 0, 0, 0, true, &lockPath)
	msb := NewMsBuilder(testDir, "mst", &lockPath, conf, 10, fileName, 0, store.Sequencer(), 2)
	write(ids, data, msb)
	fileSeq++
	store.AddTable(msb, true, false)

	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	tier1 := uint64(util.Hot)
	store = NewTableStore(testDir, &lockPath, &tier1, false, conf)
	if _, err := store.Open(); err != nil {
		t.Fatal(err)
	}
	fs := store.tableFiles("mst", true)
	if fs == nil {
		t.Fatal("get mst files fail")
	}

	f := store.File("mst", fs.Files()[0].Path(), true)
	if f == nil {
		t.Fatal("get file fail")
	}
	midx, _ := f.MetaIndexAt(0)
	if midx == nil {
		t.Fatalf("meta index not find")
	}
	decs := NewReadContext(true)
	cms, err := f.ReadChunkMetaData(0, midx, nil)
	if err != nil {
		t.Fatal(err)
	}
	rec := record.NewRecordBuilder(schema)
	fr := f.(*tsspFile).reader.(*tsspFileReader)
	fr.Unref()

	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			fr.Ref()
			_, err = f.ReadAt(&cms[0], 0, rec, decs)
			if err != nil {
				t.Error(err)
			}
			fr.Unref()
			defer wg.Done()
		}()
	}
	wg.Wait()
	if fr.ref != 0 {
		t.Fatal("ref error")
	}
}

func TestParseFileName(t *testing.T) {
	type TestCase struct {
		fName   string
		valid   bool
		seq     uint64
		level   uint16
		merge   uint16
		extent  uint16
		fmtName string
	}

	tsts := []TestCase{
		{
			fName: "00000001-0001-00010001.tssp",
			valid: true, seq: 1, level: 1, merge: 1, extent: 1,
			fmtName: "00000001-0001-00010001",
		},
		{
			fName: "100000001-0001-00010001.tssp",
			valid: true, seq: 0x100000001, level: 1, merge: 1, extent: 1,
			fmtName: "100000001-0001-00010001",
		},
		{
			fName: "000100001-0001-00010001.tssp",
			valid: true, seq: 0x100000001, level: 1, merge: 1, extent: 1,
			fmtName: "00100001-0001-00010001",
		},
		{
			fName: "0000000100000001-0001-00010001.tssp",
			valid: true, seq: 0x100000001, level: 1, merge: 1, extent: 1,
			fmtName: "100000001-0001-00010001",
		},
		{
			fName: "0000000100000001-0001-00010001.tssp.init",
			valid: true, seq: 0x100000001, level: 1, merge: 1, extent: 1,
			fmtName: "100000001-0001-00010001",
		},
		{
			fName: "00000000000000001-0001-00010001.tssp",
			valid: false, seq: 1, level: 1, merge: 1, extent: 1,
			fmtName: "00000001-0001-00010001",
		},
		{
			fName: "00000001-0001-00010001.tssp.init",
			valid: true, seq: 1, level: 1, merge: 1, extent: 1,
			fmtName: "00000001-0001-00010001",
		},
		{
			fName: "0000001a-0002-000b000f.tssp",
			valid: true, seq: 26, level: 2, merge: 11, extent: 15,
			fmtName: "0000001a-0002-000b000f",
		},
		{
			fName: "0000001a-0012-000b000f.tssp.init",
			valid: true, seq: 26, level: 18, merge: 11, extent: 15,
			fmtName: "0000001a-0012-000b000f",
		},
		{
			fName: "0000001a-0002-000b000f.tssx",
			valid: false, seq: 26, level: 2, merge: 11, extent: 15,
		},
		{
			fName: "0000001a-0002-000b000f.tssp.ini",
			valid: false, seq: 26, level: 2, merge: 11, extent: 15,
		},
		{
			fName: "00000001-0001-00010001.tssp.initt",
			valid: false, seq: 1, level: 1, merge: 1, extent: 1,
		},
	}
	dir := "/data/test/"
	for _, tst := range tsts {
		name := filepath.Join(dir, tst.fName)
		var fileName TSSPFileName
		err := fileName.ParseFileName(name)
		if !tst.valid {
			if err == nil {
				t.Fatalf("%v is a invalid file name, but check true", tst.fName)
			}
			continue
		}

		if err != nil {
			t.Fatalf("%v is a valid file name, but check false", tst.fName)
		}

		fileName.SetOrder(true)
		str := fileName.String()
		if str != tst.fmtName {
			t.Fatalf("exp:%v, get:%v", tst.fmtName, str)
		}
	}
}

func TestTsspReader(t *testing.T) {
	lockPath := ""
	fName := NewTSSPFileName(1, 0, 0, 0, false, &lockPath)
	msb := &MsBuilder{
		Conf:     NewConfig(),
		trailer:  &Trailer{},
		FileName: fName,
	}

	fd := &mockFile{
		CloseFn: func() error {
			return fmt.Errorf("close file fail")
		},
		NameFn: func() string {
			return "/tmp/0000001a-0012-000b000f.tssp.init"
		},
		StatFn: func() (os.FileInfo, error) {
			return nil, fmt.Errorf("stat fail")
		},
	}
	msb.fd = fd
	msb.trailer.bloomK, msb.trailer.bloomM = bloom.Estimate(1, falsePositive)
	msb.fileSize = 4096
	msb.bloomFilter = make([]byte, 8)

	_, err := CreateTSSPFileReader(msb.fileSize, msb.fd, msb.trailer, &msb.TableData, msb.FileVersion(), false, &lockPath)
	if err == nil || !strings.Contains(err.Error(), "table store create file failed") {
		t.Fatal("create tssp file should be fail")
	}

	fd.CloseFn = func() error { return nil }
	_, err = CreateTSSPFileReader(msb.fileSize, msb.fd, msb.trailer, &msb.TableData, msb.FileVersion(), false, &lockPath)
	if err == nil || !strings.Contains(err.Error(), "table store create file failed") {
		t.Fatal("create tssp file should be fail")
	}
}

func TestFullCompacted(t *testing.T) {
	type nameInfo struct {
		seq    uint64
		level  uint16
		extent uint16
	}
	type TestCase struct {
		exp   bool
		files []nameInfo
	}

	cases := []TestCase{
		{
			files: []nameInfo{{seq: 1, level: 3, extent: 0}},
			exp:   true,
		},
		{
			files: []nameInfo{{seq: 1, level: 0, extent: 0}},
			exp:   true,
		},

		{
			files: []nameInfo{{seq: 1, level: 3, extent: 0}, {seq: 1, level: 3, extent: 1}, {seq: 1, level: 3, extent: 2}, {seq: 1, level: 3, extent: 3}},
			exp:   true,
		},

		{
			files: []nameInfo{{seq: 1, level: 3, extent: 0}, {seq: 1, level: 3, extent: 1}, {seq: 2, level: 3, extent: 0}, {seq: 2, level: 3, extent: 1}},
			exp:   false,
		},

		{
			files: []nameInfo{{seq: 1, level: 1}, {seq: 2, level: 1}, {seq: 3, level: 1}, {seq: 4, level: 1}, {seq: 5, level: 1}},
			exp:   false,
		},
		{
			files: []nameInfo{{seq: 1, level: 0}, {seq: 2, level: 0}, {seq: 3, level: 0}, {seq: 4, level: 0}},
			exp:   false,
		},
		{
			files: []nameInfo{{seq: 1, level: 3}, {seq: 2, level: 1}, {seq: 3, level: 0}, {seq: 4, level: 0}},
			exp:   false,
		},
	}

	fs := &TSSPFiles{}
	lockPath := ""
	for _, tsc := range cases {
		fs.files = fs.files[:0]
		for _, ni := range tsc.files {
			fs.files = append(fs.files, &tsspFile{name: NewTSSPFileName(ni.seq, ni.level, 0, ni.extent, true, &lockPath)})
		}

		got := fs.fullCompacted()
		if got != tsc.exp {
			t.Fatalf("check full compacted fail, exp:%v, get:%v", tsc.exp, got)
		}
	}

}

func TestFileHandlesRef_EnableMmap(t *testing.T) {
	sig := interruptsignal.NewInterruptSignal()
	defer func() {
		sig.Close()
		fileops.RemoveAll(testDir)
	}()
	_ = fileops.RemoveAll(testDir)

	EnableMmapRead(true)
	defer EnableMmapRead(false)

	conf := NewConfig()
	tier := uint64(util.Hot)
	lockPath := ""
	store := NewTableStore(testDir, &lockPath, &tier, false, conf)
	var msb *MsBuilder
	var err error
	write := func(ids []uint64, data map[uint64]*record.Record, msb *MsBuilder) {
		for _, id := range ids {
			rec := data[id]
			err = msb.WriteData(id, rec)
			if err != nil {
				return
			}
		}
	}
	fileSeq := uint64(1)
	mst := "mst"
	var idMinMax, tmMinMax MinMax
	ids, data := genMemTableData(1, 10, 100, &idMinMax, &tmMinMax)
	fileName := NewTSSPFileName(fileSeq, 0, 0, 0, true, &lockPath)
	msb = NewMsBuilder(testDir, mst, &lockPath, conf, 10, fileName, 0, store.Sequencer(), 2)
	write(ids, data, msb)
	if err != nil {
		t.Fatal(err)
	}
	f, err := msb.NewTSSPFile(false)
	if err != nil {
		t.Fatal(err)
	}
	store.AddTSSPFiles(msb.Name(), true, f)

	err = f.FreeFileHandle()
	if err != nil {
		t.Fatal(err)
	}
	fr := f.(*tsspFile).reader.(*tsspFileReader)
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			fr.Ref()
			_, err = f.ReadData(0, 1, nil)
			if err != nil {
				t.Error(err)
			}
			fr.Unref()
			defer wg.Done()
		}()
	}
	wg.Wait()
	if fr.ref != 0 {
		t.Fatal("ref error")
	}

}

func TestCloseFileAndUnref(t *testing.T) {
	sig := interruptsignal.NewInterruptSignal()
	defer func() {
		sig.Close()
		fileops.RemoveAll(testDir)
	}()
	_ = fileops.RemoveAll(testDir)
	conf := NewConfig()
	tier := uint64(util.Hot)
	lockPath := ""
	store := NewTableStore(testDir, &lockPath, &tier, false, conf)
	var msb *MsBuilder
	var err error
	write := func(ids []uint64, data map[uint64]*record.Record, msb *MsBuilder) {
		for _, id := range ids {
			rec := data[id]
			err = msb.WriteData(id, rec)
			if err != nil {
				return
			}
		}
	}
	fileSeq := uint64(1)
	mst := "mst"
	var idMinMax, tmMinMax MinMax
	ids, data := genMemTableData(1, 10, 100, &idMinMax, &tmMinMax)
	fileName := NewTSSPFileName(fileSeq, 0, 0, 0, true, &lockPath)
	msb = NewMsBuilder(testDir, mst, &lockPath, conf, 10, fileName, 0, store.Sequencer(), 2)
	write(ids, data, msb)
	if err != nil {
		t.Fatal(err)
	}
	f, err := msb.NewTSSPFile(false)
	if err != nil {
		t.Fatal(err)
	}
	store.AddTSSPFiles(msb.Name(), true, f)

	f.RefFileReader()
	go func() {
		err := f.Close()
		if err != nil {
			t.Error(err)
		}
	}()
	go f.UnrefFileReader()
}

func TestDropMeasurement(t *testing.T) {
	sig := interruptsignal.NewInterruptSignal()
	defer func() {
		sig.Close()
		fileops.RemoveAll(testDir)
	}()
	_ = fileops.RemoveAll(testDir)
	conf := NewConfig()
	tier := uint64(util.Hot)
	lockPath := ""
	store := NewTableStore(testDir, &lockPath, &tier, false, conf)

	write := func(ids []uint64, data map[uint64]*record.Record, msb *MsBuilder) {
		for _, id := range ids {
			rec := data[id]
			err := msb.WriteData(id, rec)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	fileSeq := uint64(1)
	msts := []string{"mst", "mst1", "cpu"}
	var idMinMax, tmMinMax MinMax
	for _, mst := range msts {
		ids, data := genMemTableData(1, 10, 100, &idMinMax, &tmMinMax)
		fileName := NewTSSPFileName(fileSeq, 0, 0, 0, true, &lockPath)
		msb := NewMsBuilder(testDir, mst, &lockPath, conf, 10, fileName, 0, store.Sequencer(), 2)
		write(ids, data, msb)
		fileSeq++
		store.AddTable(msb, true, false)
	}

	files := 8 * 8
	sid := uint64(10)
	for i := 0; i < files; i++ {
		ids, data := genMemTableData(sid, 10, 100, &idMinMax, &tmMinMax)
		fileName := NewTSSPFileName(fileSeq, 0, 0, 0, true, &lockPath)
		msb := NewMsBuilder(testDir, "mst", &lockPath, conf, 10, fileName, 0, store.Sequencer(), 2)
		write(ids, data, msb)
		sid += 5
		store.AddTable(msb, true, false)
		fileSeq++
	}

	if err := store.LevelCompact(0, 1); err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 20)

	var wg sync.WaitGroup
	errs := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			if err := store.DropMeasurement(context.Background(), "mst"); err != nil {
				errs <- err
			}
			wg.Done()
		}()
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	if store.tableFiles("mst", true) != nil {
		t.Fatal("drop measurement fail")
	}

	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestClosedTsspFile(t *testing.T) {
	sig := interruptsignal.NewInterruptSignal()
	defer func() {
		sig.Close()
		fileops.RemoveAll(testDir)
	}()
	_ = fileops.RemoveAll(testDir)
	conf := NewConfig()
	tier := uint64(util.Hot)
	lockPath := ""
	store := NewTableStore(testDir, &lockPath, &tier, false, conf)

	write := func(ids []uint64, data map[uint64]*record.Record, msb *MsBuilder) {
		for _, id := range ids {
			rec := data[id]
			err := msb.WriteData(id, rec)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	fileSeq := uint64(1)
	var idMinMax, tmMinMax MinMax
	ids, data := genMemTableData(1, 10, 100, &idMinMax, &tmMinMax)
	fileName := NewTSSPFileName(fileSeq, 0, 0, 0, true, &lockPath)
	msb := NewMsBuilder(testDir, "mst", &lockPath, conf, 10, fileName, 0, store.Sequencer(), 2)
	write(ids, data, msb)
	fileSeq++
	store.AddTable(msb, true, false)

	fs := store.tableFiles("mst", true)
	if fs == nil {
		t.Fatal("get mst files fail")
	}

	fs.StopFiles()

	f := store.File("mst", fs.Files()[0].Path(), true)
	if f == nil {
		t.Fatal("get file fail")
	}

	_, _, err := f.MetaIndex(ids[0], record.TimeRange{Min: math.MinInt64, Max: math.MaxInt64})
	if err != errFileClosed {
		t.Fatal("stop fail fail")
	}

	_, err = f.MetaIndexAt(0)
	if err != errFileClosed {
		t.Fatal("stop fail fail")
	}

	var cm ChunkMeta
	_, err = f.ChunkMeta(ids[0], 0, 0, 0, 0, &cm, nil)
	if err != errFileClosed {
		t.Fatal("stop fail fail")
	}

	_, err = f.ReadData(0, 16, nil)
	if err != errFileClosed {
		t.Fatal("stop fail fail")
	}

	var mi MetaIndex
	_, err = f.ReadChunkMetaData(0, &mi, nil)
	if err != errFileClosed {
		t.Fatal("stop fail fail")
	}

	_, err = f.ReadAt(&cm, 0, nil, nil)
	if err != errFileClosed {
		t.Fatal("stop fail fail")
	}

	_, err = f.ChunkMetaAt(0)
	if err != errFileClosed {
		t.Fatal("stop fail fail")
	}

	_, err = f.Contains(ids[0])
	if err != errFileClosed {
		t.Fatal("stop fail fail")
	}
	_, err = f.ContainsValue(ids[0], record.TimeRange{Min: math.MinInt64, Max: math.MaxInt64})
	if err != errFileClosed {
		t.Fatal("stop fail fail")
	}

	_, _, err = f.MinMaxTime()
	if err != errFileClosed {
		t.Fatal("stop fail fail")
	}
}

func newMmsTables(t *testing.T) *MmsTables {
	sig := interruptsignal.NewInterruptSignal()
	defer func() {
		sig.Close()
	}()
	conf := NewConfig()
	tier := uint64(util.Hot)
	lockPath := ""
	store := NewTableStore(t.TempDir(), &lockPath, &tier, false, conf)

	write := func(ids []uint64, data map[uint64]*record.Record, msb *MsBuilder) {
		for _, id := range ids {
			rec := data[id]
			err := msb.WriteData(id, rec)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	fileSeq := uint64(1)
	var idMinMax, tmMinMax MinMax
	ids, data := genMemTableData(1, 10, 100, &idMinMax, &tmMinMax)
	fileName := NewTSSPFileName(fileSeq, 0, 0, 0, true, &lockPath)
	msb := NewMsBuilder(testDir, "mst", &lockPath, conf, 10, fileName, 0, store.Sequencer(), 2)
	write(ids, data, msb)
	fileSeq++
	store.AddTable(msb, true, false)
	return store
}

func TestLoadIdTimes(t *testing.T) {
	store := newMmsTables(t)
	seq := store.Sequencer()
	require.False(t, seq.isFree)

	count, err := store.loadIdTimesInLock()
	require.NoError(t, err, "load id times fail")
	require.Equal(t, 1000, int(count))

	idTimes, ok := seq.mmsIdTime["mst"]
	require.True(t, ok)
	require.Equal(t, 10, len(idTimes.idTime))

	for seq.IsLoading() {
		time.Sleep(100 * time.Millisecond)
		fmt.Println("seq is loading")
	}
	lastFlush, rows := idTimes.get(1)
	require.Equal(t, int64(100), rows)
	require.True(t, lastFlush > 0)
}

func TestClosedLoadIdTimes(t *testing.T) {
	store := newMmsTables(t)
	fs := store.tableFiles("mst", true)
	require.NotEmpty(t, fs, "get mst files fail")

	close(store.closed)

	count, err := store.loadIdTimesInLock()
	require.NoError(t, err, "load id times fail")
	require.Equal(t, 0, int(count))

	fr := fs.files[0].(*tsspFile).reader.(*tsspFileReader)
	require.True(t, fr.ref >= 0, "ref error")
}

func TestReadTimeColumn(t *testing.T) {
	dir := t.TempDir()
	conf := NewConfig()
	tier := uint64(util.Hot)
	lockPath := ""
	store := NewTableStore(dir, &lockPath, &tier, false, conf)
	defer store.Close()

	write := func(ids []uint64, data map[uint64]*record.Record, msb *MsBuilder) {
		for _, id := range ids {
			rec := data[id]
			err := msb.WriteData(id, rec)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	fileSeq := uint64(1)
	var idMinMax, tmMinMax MinMax
	ids, data := genMemTableData(1, 10, 100, &idMinMax, &tmMinMax)
	fileName := NewTSSPFileName(fileSeq, 0, 0, 0, true, &lockPath)
	msb := NewMsBuilder(dir, "mst", &lockPath, conf, 10, fileName, 0, store.Sequencer(), 2)
	write(ids, data, msb)
	fileSeq++
	store.AddTable(msb, true, false)

	fs := store.tableFiles("mst", true)
	if !assert.NotEmpty(t, fs, "get mst files fail") {
		return
	}

	f := store.File("mst", fs.Files()[0].Path(), true)
	if !assert.NotEmpty(t, f, "get file failed") {
		return
	}

	var cm = &ChunkMeta{}
	var err error

	midx, _ := f.MetaIndexAt(0)
	if !assert.NotEmpty(t, midx) {
		return
	}

	cm, err = f.ChunkMeta(midx.id, midx.offset, midx.size, midx.count, 0, nil, nil)
	if !assert.NoError(t, err) {
		return
	}

	dst := &record.Record{}
	cm.colMeta[0] = cm.colMeta[len(cm.colMeta)-1]
	for _, item := range cm.colMeta {
		dst.Schema = append(dst.Schema, record.Field{
			Type: int(item.ty), Name: item.name,
		})
	}
	dst.ColVals = make([]record.ColVal, len(dst.Schema))

	_, err = f.ReadAt(cm, 0, dst, &ReadContext{coderCtx: &encoding.CoderContext{}})
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, dst.Times(), dst.Column(0).IntegerValues())
}

func TestCompactionPlan(t *testing.T) {
	tier := uint64(util.Hot)
	lockPath := ""
	store := NewTableStore("", &lockPath, &tier, false, NewConfig())

	type TestCase struct {
		name         string
		tsspFileName []string
		expGroups    [][]string
	}

	cases := []TestCase{
		TestCase{
			name: "compPlan1",
			tsspFileName: []string{
				"00000001-0000-00000000.tssp",
				"00000002-0000-00000000.tssp",
				"00000003-0000-00000000.tssp",
				"00000004-0000-00000000.tssp",
				"00000005-0000-00000000.tssp",
				"00000006-0000-00000000.tssp",
				"00000007-0000-00000000.tssp",
				"00000008-0000-00000000.tssp",
				"00000009-0002-00000000.tssp",
				"00000011-0002-00000000.tssp",
				"00000019-0002-00000000.tssp",
				"00000021-0001-00000000.tssp",
			},
			expGroups: [][]string{},
		},
		TestCase{
			name: "compPlan2",
			tsspFileName: []string{
				"00000000-0000-00000000.tssp",
				"00000001-0002-00000000.tssp",
				"00000009-0002-00000000.tssp",
				"00000011-0002-00000000.tssp",
				"00000019-0002-00000000.tssp",
				"00000021-0001-00000000.tssp",
			},
			expGroups: [][]string{
				[]string{
					"00000001-0002-00000000.tssp",
					"00000009-0002-00000000.tssp",
					"00000011-0002-00000000.tssp",
					"00000019-0002-00000000.tssp",
				},
			},
		},
		TestCase{
			name: "compPlan3",
			tsspFileName: []string{
				"00000001-0002-00000000.tssp",
				"00000009-0002-00000000.tssp",
				"00000011-0002-00000000.tssp",
				"00000019-0002-00000000.tssp",
			},
			expGroups: [][]string{
				[]string{
					"00000001-0002-00000000.tssp",
					"00000009-0002-00000000.tssp",
					"00000011-0002-00000000.tssp",
					"00000019-0002-00000000.tssp",
				},
			},
		},
		TestCase{
			name: "compPlan4",
			tsspFileName: []string{
				"00000001-0002-00000000.tssp",
				"00000001-0002-00000001.tssp",
				"00000001-0002-00000002.tssp",
				"00000021-0002-00000000.tssp",
				"00000021-0002-00000001.tssp",
				"00000041-0002-00000000.tssp",
				"00000049-0002-00000000.tssp",
				"00000051-0002-00000000.tssp",
			},
			expGroups: [][]string{},
		},

		TestCase{
			name: "compPlan5",
			tsspFileName: []string{
				"00000001-0002-00000000.tssp",
				"00000001-0002-00000001.tssp",
				"00000001-0002-00000002.tssp",
				"00000021-0002-00000000.tssp",
				"00000021-0002-00000001.tssp",
				"00000041-0002-00000000.tssp",
				"00000049-0002-00000000.tssp",
				"00000051-0002-00000000.tssp",
				"00000059-0002-00000000.tssp",
			},
			expGroups: [][]string{
				[]string{
					"00000041-0002-00000000.tssp",
					"00000049-0002-00000000.tssp",
					"00000051-0002-00000000.tssp",
					"00000059-0002-00000000.tssp",
				},
			},
		},

		TestCase{
			name: "compPlan6",
			tsspFileName: []string{
				"00000001-0002-00000000.tssp",
				"00000001-0002-00000001.tssp",
				"00000001-0002-00000002.tssp",
				"00000021-0002-00000000.tssp",
				"00000021-0002-00000001.tssp",
				"00000041-0002-00000000.tssp",
				"00000049-0002-00000000.tssp",
				"00000051-0002-00000000.tssp",
				"00000059-0002-00000000.tssp",
				"00000059-0002-00000001.tssp",
			},
			expGroups: [][]string{},
		},
		TestCase{
			name: "compPlan7",
			tsspFileName: []string{
				"00000001-0002-00000000.tssp",
				"00000001-0002-00000001.tssp",
				"00000001-0002-00000002.tssp",
				"00000021-0002-00000000.tssp",
				"00000021-0002-00000001.tssp",
				"00000041-0002-00000000.tssp",
				"00000049-0002-00000000.tssp",
				"00000051-0002-00000000.tssp",
			},
			expGroups: [][]string{},
		},
		TestCase{
			name: "compPlan7",
			tsspFileName: []string{
				"00000001-0002-00000000.tssp",
				"00000001-0002-00000001.tssp",
				"00000001-0002-00000002.tssp",
				"00000021-0002-00000000.tssp",
				"00000021-0002-00000001.tssp",
				"00000041-0002-00000000.tssp",
				"00000049-0002-00000000.tssp",
				"00000049-0002-00000001.tssp",
				"00000049-0002-00000002.tssp",
				"00000051-0002-00000000.tssp",
				"00000059-0002-00000000.tssp",
				"00000061-0002-00000000.tssp",
				"00000069-0002-00000000.tssp",
				"00000071-0002-00000000.tssp",
			},
			expGroups: [][]string{
				[]string{
					"00000051-0002-00000000.tssp",
					"00000059-0002-00000000.tssp",
					"00000061-0002-00000000.tssp",
					"00000069-0002-00000000.tssp",
				},
			},
		},

		TestCase{
			name: "compPlan8",
			tsspFileName: []string{
				"00000000-0000-00000000.tssp",
				"00000001-0002-00000000.tssp",
				"00000009-0002-00000000.tssp",
				"00000011-0002-00000000.tssp",
				"00000019-0002-00000000.tssp",
				"00000021-0002-00000000.tssp",
				"00000029-0002-00000000.tssp",
				"00000029-0002-00000001.tssp",
				"00000031-0002-00000000.tssp",
				"00000039-0002-00000000.tssp",
				"00000039-0002-00000001.tssp",
				"00000041-0002-00000000.tssp",
				"00000049-0002-00000000.tssp",
				"00000051-0002-00000000.tssp",
				"00000059-0002-00000000.tssp",
				"00000061-0002-00000000.tssp",
			},
			expGroups: [][]string{
				[]string{
					"00000001-0002-00000000.tssp",
					"00000009-0002-00000000.tssp",
					"00000011-0002-00000000.tssp",
					"00000019-0002-00000000.tssp",
				},
				[]string{
					"00000041-0002-00000000.tssp",
					"00000049-0002-00000000.tssp",
					"00000051-0002-00000000.tssp",
					"00000059-0002-00000000.tssp",
				},
			},
		},
	}

	for _, c := range cases {
		for _, fn := range c.tsspFileName {
			f := genTsspFile(fn)
			if f == nil {
				t.Fatalf("parse file name (%v) fail", fn)
			}

			store.addTSSPFile(true, f, "mst")
		}

		fs := store.tableFiles("mst", true)
		plans := store.mmsPlan("mst", fs, 2, LeveLMinGroupFiles[2], nil)
		if len(plans) != len(c.expGroups) {
			t.Fatalf("exp groups :%v, get:%v", len(c.expGroups), len(plans))
		}
		for i, group := range c.expGroups {
			if !reflect.DeepEqual(group, plans[i].group) {
				t.Fatalf("exp groups :%v, get:%v", c.expGroups, plans)
			}
			store.CompactDone(group)
		}

		delete(store.Order, "mst")
	}
}

func TestCompactionPlanWithAbnormal(t *testing.T) {
	tier := uint64(util.Hot)
	lockPath := ""
	store := NewTableStore("", &lockPath, &tier, false, NewConfig())

	type TestCase struct {
		name         string
		tsspFileName []string
		expGroups    [][]string
		inCompaction []string
	}

	cases := []TestCase{
		TestCase{
			name: "mergeOutOfOrderInOtherLevel",
			tsspFileName: []string{
				"00000001-0000-00000000.tssp",
				"00000002-0000-00000000.tssp",
				"00000003-0000-00000000.tssp",
				"00000004-0000-00000000.tssp",
				"00000005-0000-00000000.tssp",
				"00000006-0000-00000000.tssp",
				"00000007-0000-00000000.tssp",
				"00000008-0000-00000000.tssp",
				"00000009-0002-00000000.tssp",
				"00000011-0002-00000000.tssp",
				"00000019-0002-00000000.tssp",
				"00000021-0002-00000000.tssp",
				"00000029-0001-00000000.tssp",
			},
			expGroups: [][]string{
				[]string{
					"00000009-0002-00000000.tssp",
					"00000011-0002-00000000.tssp",
					"00000019-0002-00000000.tssp",
					"00000021-0002-00000000.tssp",
				},
			},
			inCompaction: []string{
				"00000002-0000-00000000.tssp",
				"00000003-0000-00000000.tssp",
			},
		},
		TestCase{
			name: "mergeOutOfOrderInThisLevel",
			tsspFileName: []string{
				"00000000-0000-00000000.tssp",
				"00000001-0002-00000000.tssp",
				"00000009-0002-00000000.tssp",
				"00000011-0002-00000000.tssp",
				"00000019-0002-00000000.tssp",
				"00000021-0001-00000000.tssp",
			},
			expGroups: [][]string{},
			inCompaction: []string{
				"00000001-0002-00000000.tssp",
			},
		},
		TestCase{
			name: "mergeOutOfOrderInThisLevelV2",
			tsspFileName: []string{
				"00000001-0002-00000000.tssp",
				"00000009-0002-00000000.tssp",
				"00000011-0002-00000000.tssp",
				"00000019-0002-00000000.tssp",
				"00000021-0002-00000000.tssp",
			},
			expGroups: [][]string{},
			inCompaction: []string{
				"00000001-0002-00000000.tssp",
			},
		},
		TestCase{
			name: "mergeOutOfOrderInThisLevelV3",
			tsspFileName: []string{
				"00000001-0002-00000000.tssp",
				"00000009-0002-00000000.tssp",
				"00000011-0002-00000000.tssp",
				"00000019-0002-00000000.tssp",
				"00000021-0002-00000000.tssp",
			},
			expGroups: [][]string{},
			inCompaction: []string{
				"00000011-0002-00000000.tssp",
			},
		},
		TestCase{
			name: "lowLevelExistBeforeHighLevel",
			tsspFileName: []string{
				"00000001-0001-00000000.tssp",
				"00000002-0001-00000000.tssp",
				"00000003-0001-00000000.tssp",
				"00000004-0001-00000000.tssp",
				"00000009-0002-00000000.tssp",
				"00000011-0002-00000000.tssp",
				"00000019-0002-00000000.tssp",
				"00000021-0002-00000000.tssp",
				"00000029-0000-00000000.tssp",
			},
			expGroups: [][]string{
				[]string{
					"00000009-0002-00000000.tssp",
					"00000011-0002-00000000.tssp",
					"00000019-0002-00000000.tssp",
					"00000021-0002-00000000.tssp",
				},
			},
			inCompaction: []string{},
		},
		TestCase{
			name: "lowLevelExistBeforeHighLevelV2",
			tsspFileName: []string{
				"00000001-0002-00000000.tssp",
				"00000021-0002-00000000.tssp",
				"00000041-0001-00000000.tssp",
				"00000049-0001-00000000.tssp",
				"00000051-0001-00000000.tssp",
				"00000059-0001-00000000.tssp",
				"00000061-0002-00000000.tssp",
				"00000081-0002-00000000.tssp",
				"000000101-0002-00000000.tssp",
				"000000121-0002-00000000.tssp",
			},
			expGroups: [][]string{
				[]string{
					"00000061-0002-00000000.tssp",
					"00000081-0002-00000000.tssp",
					"000000101-0002-00000000.tssp",
					"000000121-0002-00000000.tssp",
				},
			},
			inCompaction: []string{},
		},
		TestCase{
			name: "lowLevelExistBeforeHighLevelV3",
			tsspFileName: []string{
				"00000001-0002-00000000.tssp",
				"00000021-0002-00000000.tssp",
				"00000041-0001-00000000.tssp",
				"00000049-0001-00000000.tssp",
				"00000051-0001-00000000.tssp",
				"00000059-0001-00000000.tssp",
				"00000061-0002-00000000.tssp",
				"00000081-0002-00000000.tssp",
				"000000101-0002-00000000.tssp",
				"000000121-0002-00000000.tssp",
			},
			expGroups: [][]string{},
			inCompaction: []string{
				"00000061-0002-00000000.tssp",
			},
		},
		TestCase{
			name: "mergeOutOfOrderWithSpiltFile",
			tsspFileName: []string{
				"00000001-0002-00000000.tssp",
				"00000001-0002-00000001.tssp",
				"00000001-0002-00000002.tssp",
				"00000021-0002-00000000.tssp",
				"00000021-0002-00000001.tssp",
				"00000041-0002-00000000.tssp",
				"00000049-0002-00000000.tssp",
				"00000051-0002-00000000.tssp",
			},
			expGroups: [][]string{},
			inCompaction: []string{
				"00000021-0002-00000001.tssp",
			},
		},
		TestCase{
			name: "mergeOutOfOrderWithSpiltFileV2",
			tsspFileName: []string{
				"00000001-0002-00000000.tssp",
				"00000001-0002-00000001.tssp",
				"00000001-0002-00000002.tssp",
				"00000021-0002-00000000.tssp",
				"00000021-0002-00000001.tssp",
				"00000041-0002-00000000.tssp",
				"00000049-0002-00000000.tssp",
				"00000051-0002-00000000.tssp",
				"00000059-0002-00000000.tssp",
				"00000059-0002-00000001.tssp",
			},
			expGroups: [][]string{},
			inCompaction: []string{
				"00000059-0002-00000001.tssp",
			},
		},
		TestCase{
			name: "lowLevelExistBeforeHighLevelWithSpiltFile",
			tsspFileName: []string{
				"00000001-0002-00000000.tssp",
				"00000001-0002-00000001.tssp",
				"00000001-0002-00000002.tssp",
				"00000021-0002-00000000.tssp",
				"00000021-0002-00000001.tssp",
				"00000041-0001-00000000.tssp",
				"00000049-0001-00000000.tssp",
				"00000051-0001-00000000.tssp",
				"00000059-0001-00000000.tssp",
				"00000061-0002-00000000.tssp",
				"00000081-0002-00000000.tssp",
				"000000101-0002-00000000.tssp",
			},
			expGroups:    [][]string{},
			inCompaction: []string{},
		},
	}

	for _, c := range cases {
		for _, fn := range c.tsspFileName {
			f := genTsspFile(fn)
			if f == nil {
				t.Fatalf("parse file name (%v) fail", fn)
			}

			store.addTSSPFile(true, f, "mst")
		}

		for _, fn := range c.inCompaction {
			store.inCompact[fn] = struct{}{}
		}

		fs := store.tableFiles("mst", true)
		plans := store.mmsPlan("mst", fs, 2, LeveLMinGroupFiles[2], nil)
		if len(plans) != len(c.expGroups) {
			t.Fatalf("exp groups :%v, get:%v", len(c.expGroups), len(plans))
		}
		for i, group := range c.expGroups {
			if !reflect.DeepEqual(group, plans[i].group) {
				t.Fatalf("exp groups :%v, get:%v", c.expGroups, plans)
			}
			store.CompactDone(group)
		}

		delete(store.Order, "mst")
		store.inCompact = make(map[string]struct{}, defaultCap)
	}
}

func genTsspFile(name string) TSSPFile {
	var fn TSSPFileName
	if err := fn.ParseFileName(name); err != nil {
		return nil
	}
	mr := &mockTSSPFileReader{name: name}
	mr.PathFn = func() string { return mr.name }
	mr.NameFn = func() string { return "mst" }
	return &tsspFile{
		ref:    1,
		name:   fn,
		reader: mr,
	}
}

type mockTSSPFileReader struct {
	name          string
	OpenFn        func() error
	CloseFn       func() error
	ReadAtFn      func(cm *ChunkMeta, segment int, dst *record.Record, decs *ReadContext) (*record.Record, error)
	MetaIndexAtFn func(idx int) (*MetaIndex, error)
	MetaIndexFn   func(id uint64, tr record.TimeRange) (int, *MetaIndex, error)
	ChunkMetaFn   func(id uint64, offset int64, size, itemCount uint32, metaIdx int, dst *ChunkMeta, buffer *[]byte) (*ChunkMeta, error)
	ChunkMetaAtFn func(index int) (*ChunkMeta, error)

	ReadMetaBlockFn     func(metaIdx int, id uint64, offset int64, size uint32, count uint32, dst *[]byte) ([]byte, error)
	ReadDataBlockFn     func(offset int64, size uint32, dst *[]byte) ([]byte, error)
	ReadDataFn          func(offset int64, size uint32, dst *[]byte) ([]byte, error)
	ReadChunkMetaDataFn func(metaIdx int, m *MetaIndex, dst []ChunkMeta) ([]ChunkMeta, error)
	BlockHeaderFn       func(meta *ChunkMeta, dst []record.Field) ([]record.Field, error)

	StatFn             func() *Trailer
	MinMaxSeriesIDFn   func() (min, max uint64, err error)
	MinMaxTimeFn       func() (min, max int64, err error)
	ContainsFn         func(id uint64, tm record.TimeRange) bool
	ContainsTimeFn     func(tm record.TimeRange) bool
	ContainsIdFn       func(id uint64) bool
	CreateTimeFn       func() int64
	NameFn             func() string
	PathFn             func() string
	RenameFn           func(newName string) error
	FileSizeFn         func() int64
	InMemSizeFn        func() int64
	VersionFn          func() uint64
	FreeMemoryFn       func() int64
	LoadIntoMemoryFn   func() error
	LoadComponentsFn   func() error
	AverageChunkRowsFn func() int
	MaxChunkRowsFn     func() int
	FreeFileHandleFn   func() error
	RefFn              func()
	UnrefFn            func()
}

func (r *mockTSSPFileReader) Open() error  { return r.OpenFn() }
func (r *mockTSSPFileReader) Close() error { return r.CloseFn() }
func (r *mockTSSPFileReader) ReadAt(cm *ChunkMeta, segment int, dst *record.Record, decs *ReadContext) (*record.Record, error) {
	return r.ReadAtFn(cm, segment, dst, decs)
}
func (r *mockTSSPFileReader) MetaIndexAt(idx int) (*MetaIndex, error) { return r.MetaIndexAtFn(idx) }
func (r *mockTSSPFileReader) MetaIndex(id uint64, tr record.TimeRange) (int, *MetaIndex, error) {
	return r.MetaIndexFn(id, tr)
}
func (r *mockTSSPFileReader) ChunkMeta(id uint64, offset int64, size, itemCount uint32, metaIdx int, dst *ChunkMeta, buffer *[]byte) (*ChunkMeta, error) {
	return r.ChunkMetaFn(id, offset, size, itemCount, metaIdx, dst, buffer)
}
func (r *mockTSSPFileReader) ChunkMetaAt(index int) (*ChunkMeta, error) {
	return r.ChunkMetaAtFn(index)
}

func (r *mockTSSPFileReader) ReadMetaBlock(metaIdx int, id uint64, offset int64, size uint32, count uint32, dst *[]byte) ([]byte, error) {
	return r.ReadMetaBlockFn(metaIdx, id, offset, size, count, dst)
}
func (r *mockTSSPFileReader) ReadDataBlock(offset int64, size uint32, dst *[]byte) ([]byte, error) {
	return r.ReadDataBlockFn(offset, size, dst)
}
func (r *mockTSSPFileReader) ReadData(offset int64, size uint32, dst *[]byte) ([]byte, error) {
	return r.ReadDataFn(offset, size, dst)
}
func (r *mockTSSPFileReader) ReadChunkMetaData(metaIdx int, m *MetaIndex, dst []ChunkMeta) ([]ChunkMeta, error) {
	return r.ReadChunkMetaDataFn(metaIdx, m, dst)
}
func (r *mockTSSPFileReader) BlockHeader(meta *ChunkMeta, dst []record.Field) ([]record.Field, error) {
	return r.BlockHeaderFn(meta, dst)
}

func (r *mockTSSPFileReader) FileStat() *Trailer { return r.StatFn() }
func (r *mockTSSPFileReader) MinMaxSeriesID() (min, max uint64, err error) {
	return r.MinMaxSeriesIDFn()
}
func (r *mockTSSPFileReader) MinMaxTime() (min, max int64, err error) { return r.MinMaxTimeFn() }
func (r *mockTSSPFileReader) ContainsValue(id uint64, tm record.TimeRange) (bool, error) {
	return r.ContainsFn(id, tm), nil
}
func (r *mockTSSPFileReader) ContainsTime(tm record.TimeRange) (bool, error) {
	return r.ContainsTimeFn(tm), nil
}
func (r *mockTSSPFileReader) Contains(id uint64) (bool, error) { return r.ContainsIdFn(id), nil }
func (r *mockTSSPFileReader) CreateTime() int64                { return r.CreateTimeFn() }
func (r *mockTSSPFileReader) Name() string                     { return r.NameFn() }
func (r *mockTSSPFileReader) Path() string                     { return r.PathFn() }
func (r *mockTSSPFileReader) Rename(newName string) error      { return r.RenameFn(newName) }
func (r *mockTSSPFileReader) FileSize() int64                  { return r.FileSizeFn() }
func (r *mockTSSPFileReader) InMemSize() int64                 { return r.InMemSizeFn() }
func (r *mockTSSPFileReader) Version() uint64                  { return r.VersionFn() }
func (r *mockTSSPFileReader) FreeMemory() int64 {
	if r.FreeMemoryFn == nil {
		return 0
	}
	return r.FreeMemoryFn()
}
func (r *mockTSSPFileReader) LoadIntoMemory() error { return r.LoadIntoMemoryFn() }
func (r *mockTSSPFileReader) LoadComponents() error { return r.LoadComponentsFn() }
func (r *mockTSSPFileReader) AverageChunkRows() int { return r.AverageChunkRowsFn() }
func (r *mockTSSPFileReader) MaxChunkRows() int     { return r.MaxChunkRowsFn() }
func (r *mockTSSPFileReader) FreeFileHandle() error { return r.FreeFileHandleFn() }
func (r *mockTSSPFileReader) Ref()                  {}
func (r *mockTSSPFileReader) Unref()                {}

func TestCompareFile(t *testing.T) {
	var setMinMax = func(f TSSPFile, min, max int64) {
		tmp := f.(*tsspFile)
		rd := tmp.reader.(*mockTSSPFileReader)
		rd.MinMaxTimeFn = func() (int64, int64, error) {
			return min, max, nil
		}
	}

	f1 := genTsspFile("00000001-0000-00000000.tssp")
	f2 := genTsspFile("00000002-0000-00000000.tssp")
	f3 := genTsspFile("00000003-0000-00000000.tssp")
	f4 := genTsspFile("00000004-0000-00000000.tssp")
	setMinMax(f1, 10, 20)
	setMinMax(f2, 10, 30)
	setMinMax(f3, 15, 18)
	setMinMax(f4, 13, 20)

	assert.True(t, compareFile(f1, f2))
	assert.True(t, !compareFile(f1, f3))
	assert.True(t, compareFile(f3, f2))

	assert.True(t, compareFileByDescend(f1, f4))
	assert.True(t, compareFileByDescend(f2, f4))
	assert.True(t, !compareFileByDescend(f3, f4))
}

func TestFreeMemory(t *testing.T) {
	f1 := genTsspFile("00000001-0000-00000000.tssp")

	timer := time.NewTimer(3 * time.Second)
	signal := make(chan struct{})

	go func() {
		defer close(signal)

		f1.Ref()
		require.Equal(t, int64(0), f1.Free(true))
		f1.Unref()
	}()

	select {
	case <-signal:
		break
	case <-timer.C:
		t.Fatalf("failed to FreeMemory")
	}
}

func TestReadError(t *testing.T) {
	EnableReadCache(102400)
	en := mmapEn
	EnableMmapRead(false)

	defer func() {
		EnableMmapRead(en)
		EnableReadCache(0)
	}()

	dir := t.TempDir()
	conf := NewConfig()
	tier := uint64(util.Hot)
	lockPath := ""
	store := NewTableStore(dir, &lockPath, &tier, false, conf)

	write := func(ids []uint64, data map[uint64]*record.Record, msb *MsBuilder) {
		for _, id := range ids {
			require.NoError(t, msb.WriteData(id, data[id]))
		}
	}

	fileSeq := uint64(1)
	var idMinMax, tmMinMax MinMax
	ids, data := genMemTableData(1, 1, 10, &idMinMax, &tmMinMax)
	fileName := NewTSSPFileName(fileSeq, 0, 0, 0, true, &lockPath)
	msb := NewMsBuilder(dir, "mst", &lockPath, conf, 10, fileName, 0, store.Sequencer(), 2)
	write(ids, data, msb)
	fileSeq++
	store.AddTable(msb, true, false)

	fs := store.tableFiles("mst", true)
	require.NotEmpty(t, fs)

	defer fs.StopFiles()

	f := store.File("mst", fs.Files()[0].Path(), true)
	require.NotEmpty(t, f)
	defer f.Close()

	tf, ok := f.(*tsspFile)
	require.True(t, ok)

	var err error
	buf := make([]byte, 0, 1000)
	buf, err = f.ReadData(0, 2000, &buf)
	require.NotEmpty(t, err)

	_, err = tf.reader.ReadDataBlock(0, 2000, &buf)
	require.NotEmpty(t, err)

	_, err = tf.reader.ReadDataBlock(0, 2000, &buf)
	require.NotEmpty(t, err)
}
