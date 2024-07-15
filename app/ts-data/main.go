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

package main

import (
	"os"

	"github.com/openGemini/openGemini/app"
	ingestserver "github.com/openGemini/openGemini/app/ts-sql/sql"
	store "github.com/openGemini/openGemini/app/ts-store/run"
	"github.com/openGemini/openGemini/lib/config"
	"github.com/openGemini/openGemini/lib/errno"
)

func main() {
	app.InitParse()
	errno.SetNode(errno.NodeData)

	info := app.ServerInfo{
		App:       config.AppData,
		Version:   app.Version,
		Commit:    app.GitCommit,
		Branch:    app.GitBranch,
		BuildTime: app.BuildTime,
	}

	cmdStore := store.NewCommand(info, true)
	cmdSql := ingestserver.NewCommand(info, false)
	cmdStore.Logo = app.TSDATALOGO
	cmdSql.Logo = ""

	app.Run(os.Args[1:], cmdStore, cmdSql)
}
