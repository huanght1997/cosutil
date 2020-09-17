/*
Copyright © 2020 Haitao Huang <hht970222@gmail.com>

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
package cli

import (
	"context"
	"net/http"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	log "github.com/sirupsen/logrus"
)

func (client *Client) InfoObject(cosPath string, _ bool) bool {
	resp, err := client.Client.Object.Head(context.Background(), cosPath, nil)
	if resp != nil && resp.StatusCode != 200 {
		log.Warnf("Head Object Response Code: %d, headers: %v",
			resp.StatusCode, resp.Header)
		return false
	} else if err != nil {
		log.Warn(err.Error())
		return false
	}
	printInfo(&resp.Header, cosPath)
	return true
}

func printInfo(header *http.Header, cosPath string) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateColumns = false

	t.AppendRow(table.Row{"Key", cosPath})
	for key, value := range *header {
		for _, v := range value {
			t.AppendRow(table.Row{key, v})
		}
	}
	t.Render()
}
