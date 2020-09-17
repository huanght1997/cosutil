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
	"io/ioutil"
	"strings"

	"github.com/huanght1997/cos-go-sdk-v5"
	log "github.com/sirupsen/logrus"
)

type GrantOption struct {
	Id         string
	Permission string
}

func (client *Client) PutObjectAcl(grantRead, grantWrite, grantFullControl, cosPath string) bool {
	var acl []GrantOption
	if grantRead != "" {
		for _, u := range strings.Split(grantRead, ",") {
			if u != "" {
				acl = append(acl, GrantOption{u, "READ"})
			}
		}
	}
	if grantWrite != "" {
		for _, u := range strings.Split(grantWrite, ",") {
			if u != "" {
				acl = append(acl, GrantOption{u, "WRITE"})
			}
		}
	}
	if grantFullControl != "" {
		for _, u := range strings.Split(grantFullControl, ",") {
			if u != "" {
				acl = append(acl, GrantOption{u, "FULL_CONTROL"})
			}
		}
	}
	result, resp, err := client.Client.Object.GetACL(context.Background(), cosPath)
	if resp != nil && resp.StatusCode != 200 {
		respContent, _ := ioutil.ReadAll(resp.Body)
		log.Warnf("GetObjectACL Response Code: %d, Response Content: %s",
			resp.StatusCode, string(respContent))
		return false
	} else if err != nil {
		log.Warnf(err.Error())
		return false
	} else {
		ownerId := result.Owner.ID
		subid, rootid, accountType := "", "", ""
		var accessControlList []cos.ACLGrant
		for _, o := range acl {
			idSeg := strings.Split(o.Id, "/")
			switch len(idSeg) {
			case 1:
				accountType = "RootAccount"
				rootid = idSeg[0]
				subid = idSeg[0]
			case 2:
				accountType = "SubAccount"
				rootid = idSeg[0]
				subid = idSeg[1]
			default:
				log.Warn("ID format error!")
				return false
			}
			id := ""
			if subid != "anyone" {
				if subid == rootid {
					id = rootid
				} else {
					id = rootid + "/" + subid
				}
			} else {
				id = "qcs::cam::anyone::anyone"
			}
			accessControlList = append(accessControlList, cos.ACLGrant{
				Grantee: &cos.ACLGrantee{
					Type: accountType,
					ID:   id,
				},
				Permission: o.Permission,
			})
		}
		option := &cos.ObjectPutACLOptions{
			Body: &cos.ACLXml{
				Owner: &cos.Owner{
					ID: ownerId,
				},
				AccessControlList: accessControlList,
			},
		}
		resp, err = client.Client.Object.PutACL(context.Background(), cosPath, option)
		if resp != nil && resp.StatusCode != 200 {
			respContent, _ := ioutil.ReadAll(resp.Body)
			log.Debug(resp.Header)
			log.Warnf("GetObjectACL Response Code: %d, Response Content: %s",
				resp.StatusCode, string(respContent))
			return false
		} else if err != nil {
			log.Warnf(err.Error())
			return false
		} else {
			log.Debug(resp.Header)
			return true
		}
	}
}
