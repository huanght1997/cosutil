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

type grantOption struct {
	ID         string
	Permission string
}

// PutObjectACL reads the grant strings and applied them to cosPath specified.
// If everything goes ok, return true.
// Otherwise(like network connection failed, no such remote object), return false.
func (client *Client) PutObjectACL(grantRead, grantWrite, grantFullControl, cosPath string) bool {
	acl := client.initACL(grantRead, grantWrite, grantFullControl)

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
		ownerID := result.Owner.ID
		return client.putACL(acl, ownerID, cosPath)
	}
}

// PutBucketACL reads the grant strings and applied them to bucketPath specified.
// If everything goes ok, return true.
// Otherwise(like network connection failed, no such remote object), return false.
func (client *Client) PutBucketACL(grantRead, grantWrite, grantFullControl, cosPath string) bool {
	acl := client.initACL(grantRead, grantWrite, grantFullControl)
	result, resp, err := client.Client.Bucket.GetACL(context.Background())
	if resp != nil && resp.StatusCode != 200 {
		respContent, _ := ioutil.ReadAll(resp.Body)
		log.Warnf("GetBucketACL Response Code: %d, Response Content: %s",
			resp.StatusCode, string(respContent))
		return false
	} else if err != nil {
		log.Warnf(err.Error())
		return false
	} else {
		ownerID := result.Owner.ID
		return client.putACL(acl, ownerID, cosPath)
	}
}

func (client *Client) initACL(grantRead, grantWrite, grantFullControl string) (acl []grantOption) {
	if grantRead != "" {
		for _, u := range strings.Split(grantRead, ",") {
			if u != "" {
				acl = append(acl, grantOption{u, "READ"})
			}
		}
	}
	if grantWrite != "" {
		for _, u := range strings.Split(grantWrite, ",") {
			if u != "" {
				acl = append(acl, grantOption{u, "WRITE"})
			}
		}
	}
	if grantFullControl != "" {
		for _, u := range strings.Split(grantFullControl, ",") {
			if u != "" {
				acl = append(acl, grantOption{u, "FULL_CONTROL"})
			}
		}
	}
	return
}

func (client *Client) putACL(acl []grantOption, ownerID string, cosPath string) bool {
	subid, rootid, accountType := "", "", ""
	var accessControlList []cos.ACLGrant
	for _, o := range acl {
		idSeg := strings.Split(o.ID, "/")
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
				ID: ownerID,
			},
			AccessControlList: accessControlList,
		},
	}
	resp, err := client.Client.Object.PutACL(context.Background(), cosPath, option)
	if resp != nil && resp.StatusCode != 200 {
		respContent, _ := ioutil.ReadAll(resp.Body)
		log.Debug(resp.Header)
		log.Warnf("PutObjectACL Response Code: %d, Response Content: %s",
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