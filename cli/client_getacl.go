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
	"fmt"
	"io/ioutil"
	"os"

	"github.com/huanght1997/cos-go-sdk-v5"
	"github.com/jedib0t/go-pretty/v6/table"
	log "github.com/sirupsen/logrus"
)

// GetBucketACL will get the ACL configured in client and prints it.
// if all things goes smoothly while requesting the information, return true.
// Otherwise, return false.
func (client *Client) GetBucketACL() bool {
	result, resp, err := client.Client.Bucket.GetACL(context.Background())
	if resp != nil && resp.StatusCode != 200 {
		respContent, _ := ioutil.ReadAll(resp.Body)
		log.Warnf("Get Bucket ACL Response Code: %d, Response Content: %s",
			resp.StatusCode, string(respContent))
		return false
	} else if err != nil {
		log.Warn(err.Error())
		return false
	} else {
		printACL(client.Config.Bucket, result.AccessControlList)
		return true
	}
}

func (client *Client) GetObjectACL(cosPath string) bool {
	result, resp, err := client.Client.Object.GetACL(context.Background(), cosPath)
	if resp != nil && resp.StatusCode != 200 {
		respContent, _ := ioutil.ReadAll(resp.Body)
		log.Warnf("Get Object ACL Response Code: %d, Response Content: %s",
			resp.StatusCode, string(respContent))
		return false
	} else if err != nil {
		log.Warn(err.Error())
		return false
	} else {
		printACL(cosPath, result.AccessControlList)
		return true
	}
}

func printACL(path string, acl []cos.ACLGrant) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendRow(table.Row{path, path}, table.RowConfig{AutoMerge: true})
	t.AppendSeparator()
	for _, grant := range acl {
		id := ""
		if grant.Grantee == nil || grant.Grantee.ID == "" {
			id = "anyone"
		} else {
			id = grant.Grantee.ID
		}
		t.AppendRow(table.Row{
			"ACL",
			fmt.Sprintf("%s: %s", id, grant.Permission),
		})
	}
	t.Render()
}
