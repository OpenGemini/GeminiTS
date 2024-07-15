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
package immutable

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"sort"
	"strings"

	obs2 "github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"github.com/openGemini/openGemini/engine/immutable/colstore"
	"github.com/openGemini/openGemini/lib/fileops"
	"github.com/openGemini/openGemini/lib/logger"
	"github.com/openGemini/openGemini/lib/obs"
	"github.com/openGemini/openGemini/lib/request"
	"github.com/openGemini/openGemini/lib/util"
	"go.uber.org/zap"
)

const (
	MetaIndexLimitNum         = 16
	MetaIndexHeaderSize int64 = 16
	MetaIndexItemSize         = int64(util.Int64SizeBytes*3 + util.Uint64SizeBytes + util.Uint32SizeBytes)
)

type DetachedMetaIndexReader struct {
	r fileops.BasicFileReader
}

func NewDetachedMetaIndexReader(path string, obsOpts *obs.ObsOptions) (*DetachedMetaIndexReader, error) {
	fd, err := fileops.OpenObsFile(path, MetaIndexFile, obsOpts, true)
	if err != nil {
		return nil, err
	}
	dr := fileops.NewFileReader(fd, nil)
	return &DetachedMetaIndexReader{r: dr}, nil
}

func (reader *DetachedMetaIndexReader) ReadMetaIndex(offset, length []int64) ([]*MetaIndex, error) {
	c := make(chan *request.StreamReader, 1)
	reader.r.StreamReadBatch(offset, length, c, MetaIndexLimitNum, true)
	metaIndexs := make([]*MetaIndex, len(offset))
	i := 0
	for r := range c {
		if r.Err != nil {
			return nil, r.Err
		}
		metaIndexs[i] = &MetaIndex{}
		if len(r.Content) < crcSize {
			return nil, fmt.Errorf("get wrong data")
		}
		if binary.BigEndian.Uint32(r.Content[:crcSize]) != crc32.ChecksumIEEE(r.Content[crcSize:]) {
			return nil, fmt.Errorf("get wrong data")
		}
		r.Content = r.Content[crcSize:]
		_, err := metaIndexs[i].unmarshalDetached(r.Content)
		if err != nil {
			return nil, err
		}
		i += 1
	}
	sort.Slice(metaIndexs, func(i, j int) bool {
		return metaIndexs[i].offset < metaIndexs[j].offset
	})
	return metaIndexs, nil
}

func (reader *DetachedMetaIndexReader) Close() {
	if reader.r != nil {
		reader.r.Close()
	}
}

func GetMetaIndexAndBlockId(path string, obsOpts *obs.ObsOptions, chunkCount int64, tr util.TimeRange) ([]int64, []*MetaIndex, error) {
	var miChunkIds []int64
	var miFiltered []*MetaIndex
	metaIndexReader, err := NewDetachedMetaIndexReader(path, obsOpts)
	if err != nil {
		return nil, nil, err
	}
	startChunkId, endChunkId := int64(0), chunkCount
	defer metaIndexReader.Close()

	// step1: init the meta index
	offsets, lengths := make([]int64, 0, chunkCount), make([]int64, 0, chunkCount)
	for i := startChunkId; i < endChunkId; i++ {
		offset, length := GetMetaIndexOffsetAndLengthByChunkId(i)
		offsets, lengths = append(offsets, offset), append(lengths, length)
	}
	metaIndexes, err := metaIndexReader.ReadMetaIndex(offsets, lengths)
	if err != nil {
		return nil, nil, err
	}
	for i := range metaIndexes {
		if metaIndexes[i].IsExist(tr) {
			miFiltered = append(miFiltered, metaIndexes[i])
			miChunkIds = append(miChunkIds, startChunkId+int64(i))
		}
	}
	return miChunkIds, miFiltered, nil
}

func GetPKItems(path string, obsOpts *obs.ObsOptions, miChunkIds []int64) (*colstore.DetachedPKMetaInfo, []*colstore.DetachedPKInfo, error) {
	pkMetaInfo, err := ReadPKMetaInfoAll(path, obsOpts)
	if err != nil {
		return nil, nil, err
	}

	offsets, lengths := make([]int64, 0, len(miChunkIds)), make([]int64, 0, len(miChunkIds))
	for _, chunkId := range miChunkIds {
		offset, length := GetPKMetaOffsetLengthByChunkId(pkMetaInfo, int(chunkId))
		offsets, lengths = append(offsets, offset), append(lengths, length)
	}
	pkMetas, err := ReadPKMetaAll(path, obsOpts, offsets, lengths)
	if err != nil {
		return nil, nil, err
	}
	offsets, lengths = offsets[:0], lengths[:0]
	for i := range pkMetas {
		offsets, lengths = append(offsets, int64(pkMetas[i].Offset)), append(lengths, int64(pkMetas[i].Length))
	}
	pkDatas, err := ReadPKDataAll(path, obsOpts, offsets, lengths, pkMetas, pkMetaInfo)
	if err != nil {
		return nil, nil, err
	}
	var pkItems []*colstore.DetachedPKInfo
	for i := range pkDatas {
		pkItems = append(pkItems, colstore.GetPKInfoByPKMetaData(pkMetas[i], pkDatas[i], pkMetaInfo.TCLocation))
	}
	return pkMetaInfo, pkItems, nil
}

func GetMetaIndexOffsetAndLengthByChunkId(chunkId int64) (offset, length int64) {
	return chunkId*(crcSize+MetaIndexItemSize) + MetaIndexHeaderSize, crcSize + MetaIndexItemSize
}

func GetMetaIndexChunkCount(obsOptions *obs.ObsOptions, dataPath string) (int64, error) {
	fd, err := fileops.OpenObsFile(dataPath, MetaIndexFile, obsOptions, true)
	if err != nil {
		obsErr, ok := err.(obs2.ObsError)
		if (ok && obsErr.StatusCode == 404) || strings.Contains(err.Error(), "no such file or directory") {
			logger.GetLogger().Error("obs dir not exist", zap.Error(err))
			return 0, nil
		}
		return 0, err
	}
	defer fd.Close()
	miFileInfo, err := fd.Stat()
	if err != nil {
		return 0, err
	}
	miFileSize := miFileInfo.Size()
	return (miFileSize - MetaIndexHeaderSize) / (crcSize + MetaIndexItemSize), nil
}
