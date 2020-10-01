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
	"strings"

	"github.com/huanght1997/cos-go-sdk-v5"
	log "github.com/sirupsen/logrus"
)

func (client *Client) CreateBucket() bool {
	_, err := client.Client.Bucket.Put(context.Background(), nil)
	if err != nil {
		r, ok := err.(*cos.ErrorResponse)
		if ok {
			log.Warnf(fmt.Sprintf("%v %s: %d %v(Message: %v)",
				r.Response.Request.Method, strings.ReplaceAll(r.Response.Request.URL.String(), "%", "%%"),
				r.Response.StatusCode, r.Code, r.Message))
		} else {
			log.Warn(err.Error())
		}
		return false
	} else {
		log.Infof("Create cos://%s", client.Config.Bucket)
		return true
	}
}
