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

	log "github.com/sirupsen/logrus"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func (client *Client) PutBucketVersioning(versioning bool) bool {
	status := ""
	if versioning {
		status = "Enabled"
	} else {
		status = "Suspended"
	}
	_, err := client.Client.Bucket.PutVersioning(context.Background(), &cos.BucketPutVersionOptions{
		Status: status,
	})
	if err != nil {
		log.Warnf(err.Error())
		return false
	} else {
		return true
	}
}
