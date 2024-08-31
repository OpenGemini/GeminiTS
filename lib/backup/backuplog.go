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

package backup

type BackupLogInfo struct {
	FullBackupTime        int64                 `json:"fullBackupTime"`
	IncBackupTime         int64                 `json:"incBackupTime"`
	OrderFileListMap      map[string][][]string `json:"orderFileListMap"`
	OutOfOrderFileListMap map[string][][]string `json:"outOfOrderFileListMap"`
}

type IncBackupLogInfo struct {
	AddOrderFileListMap      map[string][][]string `json:"addOrderFileListMap"`
	DelOrderFileListMap      map[string][][]string `json:"delOrderFileListMap"`
	AddOutOfOrderFileListMap map[string][][]string `json:"addOutOfOrderFileListMap"`
	DelOutOfOrderFileListMap map[string][][]string `json:"delOutOfOrderFileListMap"`
}

type MetaBackupLogInfo struct {
	MetaIds []string `json:"metaIds"`
	IsNode  bool     `json:"isNode"`
}
