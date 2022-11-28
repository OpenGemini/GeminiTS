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
package clvIndex

import (
	"github.com/openGemini/openGemini/lib/mpTrie"
	"sync"
	"time"

	"github.com/openGemini/openGemini/lib/utils"
	"github.com/openGemini/openGemini/lib/vGram/gramIndex"
	"github.com/openGemini/openGemini/lib/vToken/tokenIndex"
)

/*
	SHARDBUFFER is the number of data of a SHARD, LogIndex is a counter, and BuffLogStrings is used to store all the data of a SHARD, which is used to build indexes in batches.
*/
const SHARDBUFFER = 500000
const UPDATE_INTERVAL time.Duration = 30
const INDEX_PERSISTENCE_INTERVAL time.Duration = time.Hour
const INDEXOUTPATH = "../../lib/persistence/" //clvTable/logs/VGRAM/index/

type semaphore int

const (
	update semaphore = iota
	close
)

type CLVIndexNode struct {
	measurementAndFieldKey MeasurementAndFieldKey
	dicType                CLVDicType
	dic                    *CLVDictionary
	indexType              CLVIndexType
	VgramIndexRoot         *gramIndex.IndexTree
	LogTreeRoot            *gramIndex.LogTree
	VtokenIndexRoot        *tokenIndex.IndexTree

	dataSignal chan semaphore
	dataLock   sync.Mutex
	dataLen    int
	dataBuf    []utils.LogSeries
}

func NewCLVIndexNode(indexType CLVIndexType, dic *CLVDictionary, measurementAndFieldKey MeasurementAndFieldKey) *CLVIndexNode {
	clvIndex := &CLVIndexNode{
		measurementAndFieldKey: measurementAndFieldKey,
		dicType:                CLVC,
		dic:                    dic,
		indexType:              indexType,
		VgramIndexRoot:         gramIndex.NewIndexTree(QMINGRAM, QMAXGRAM),
		LogTreeRoot:            gramIndex.NewLogTree(QMAXGRAM),
		VtokenIndexRoot:        tokenIndex.NewIndexTree(QMINTOKEN, QMAXTOKEN),
		dataSignal:             make(chan semaphore),
		dataBuf:                make([]utils.LogSeries, 0, SHARDBUFFER),
	}
	clvIndex.Open()
	clvIndex.Flush()
	return clvIndex
}

var LogIndex = 0

func (clvIndex *CLVIndexNode) Open() {
	go clvIndex.updateClvIndexRoutine()
}

func (clvIndex *CLVIndexNode) Close() {
	clvIndex.dataSignal <- close
}

func (clvIndex *CLVIndexNode) updateClvIndexRoutine() {
	timer := time.NewTimer(UPDATE_INTERVAL * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-timer.C: //update the index tree periodically
			clvIndex.updateClvIndex()
		case signal, ok := <-clvIndex.dataSignal:
			if !ok {
				return
			}
			if signal == close {
				return
			}
			clvIndex.updateClvIndex()
		}
	}
}

func (clvIndex *CLVIndexNode) updateClvIndex() {
	var logbuf []utils.LogSeries

	clvIndex.dataLock.Lock()
	if clvIndex.dataLen == 0 {
		clvIndex.dataLock.Unlock()
		return
	}
	logbuf = clvIndex.dataBuf
	clvIndex.dataBuf = make([]utils.LogSeries, 0, SHARDBUFFER)
	clvIndex.dataLen = 0
	clvIndex.dataLock.Unlock()

	if clvIndex.indexType == VGRAM {
		clvIndex.CreateCLVVGramIndexIfNotExists(logbuf)
	} else if clvIndex.indexType == VTOKEN {
		clvIndex.CreateCLVVTokenIndexIfNotExists(logbuf)
	}
	LogIndex = 0
}

func (clvIndex *CLVIndexNode) CreateCLVIndexIfNotExists(log string, tsid uint64, timeStamp int64) {
	if LogIndex < SHARDBUFFER {
		clvIndex.dataBuf = append(clvIndex.dataBuf, utils.LogSeries{Log: log, Tsid: tsid, TimeStamp: timeStamp})
		LogIndex += 1
	}
	if LogIndex >= SHARDBUFFER {
		clvIndex.dataSignal <- update
	}
}

func (clvIndexNode *CLVIndexNode) CreateCLVVGramIndexIfNotExists(buffLogStrings []utils.LogSeries) {
	if clvIndexNode.dicType == CLVC {
		clvIndexNode.VgramIndexRoot, _, clvIndexNode.LogTreeRoot = gramIndex.AddIndex(buffLogStrings, QMINGRAM, QMAXGRAM, LOGTREEMAX, clvIndexNode.dic.VgramDicRoot.Root(), clvIndexNode.LogTreeRoot, clvIndexNode.VgramIndexRoot)
	}
	if clvIndexNode.dicType == CLVL {
		clvIndexNode.VgramIndexRoot, _, clvIndexNode.LogTreeRoot = gramIndex.AddIndex(buffLogStrings, QMINGRAM, clvIndexNode.dic.VgramDicRoot.Qmax(), LOGTREEMAX, clvIndexNode.dic.VgramDicRoot.Root(), clvIndexNode.LogTreeRoot, clvIndexNode.VgramIndexRoot)
	}
}

func (clvIndexNode *CLVIndexNode) CreateCLVVTokenIndexIfNotExists(buffLogStrings []utils.LogSeries) {
	if clvIndexNode.dicType == CLVC {
		clvIndexNode.VtokenIndexRoot, _ = tokenIndex.AddIndex(buffLogStrings, QMINTOKEN, QMAXTOKEN, clvIndexNode.dic.VtokenDicRoot.Root(), clvIndexNode.VtokenIndexRoot)
	}
	if clvIndexNode.dicType == CLVL {
		clvIndexNode.VtokenIndexRoot, _ = tokenIndex.AddIndex(buffLogStrings, QMINTOKEN, clvIndexNode.dic.VtokenDicRoot.Qmax(), clvIndexNode.dic.VtokenDicRoot.Root(), clvIndexNode.VtokenIndexRoot)
	}
}

func (clvIndexNode *CLVIndexNode) Flush() {
	go clvIndexNode.flushClvIndexRoutine()
}

func (clvIndexNode *CLVIndexNode) flushClvIndexRoutine() {
	timer := time.NewTimer(INDEX_PERSISTENCE_INTERVAL)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			clvIndexNode.serializeIndex()
		}
	}
}

func (clvIndexNode *CLVIndexNode) serializeIndex() { //clvTable/logs/VGRAM/index/
	if clvIndexNode.indexType == VGRAM {
		outPath := INDEXOUTPATH + clvIndexNode.measurementAndFieldKey.measurementName + "/" + clvIndexNode.measurementAndFieldKey.fieldKey + "/" + "VGRAM/" + "index/" + "index0.txt"
		indexTree := clvIndexNode.VgramIndexRoot
		mpTrie.SerializeGramIndexToFile(indexTree, outPath)
		clvIndexNode.VgramIndexRoot = gramIndex.NewIndexTree(QMINGRAM, QMAXGRAM)
		logTree := clvIndexNode.LogTreeRoot
		mpTrie.SerializeLogTreeToFile(logTree, INDEXOUTPATH+clvIndexNode.measurementAndFieldKey.measurementName+"/"+clvIndexNode.measurementAndFieldKey.fieldKey+"/"+"VGRAM/"+"logTree/"+"log0.txt")
		clvIndexNode.LogTreeRoot = gramIndex.NewLogTree(QMAXGRAM)
	}
	if clvIndexNode.indexType == VTOKEN {
		outPath := INDEXOUTPATH + clvIndexNode.measurementAndFieldKey.measurementName + "/" + clvIndexNode.measurementAndFieldKey.fieldKey + "/" + "VTOKEN/" + "index/" + "index0.txt"
		indexTree := clvIndexNode.VtokenIndexRoot
		mpTrie.SerializeTokenIndexToFile(indexTree, outPath)
		clvIndexNode.VtokenIndexRoot = tokenIndex.NewIndexTree(QMINGRAM, QMAXGRAM)
	}
}
