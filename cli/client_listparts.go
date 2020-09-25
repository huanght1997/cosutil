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

	"github.com/huanght1997/cos-go-sdk-v5"
	log "github.com/sirupsen/logrus"
)

func (client *Client) ListMultipartObjects(cosPath string) bool {
	log.Debug("Getting uploaded parts")
	keyMarker := ""
	uploadIDMarker := ""
	isTruncated := true
	partNum := 0
	for isTruncated {
		isTruncated = false
		result, resp, err := client.Client.Bucket.ListMultipartUploads(context.Background(), &cos.ListMultipartUploadsOptions{
			Delimiter:      "",
			Prefix:         cosPath,
			MaxUploads:     10,
			KeyMarker:      keyMarker,
			UploadIDMarker: uploadIDMarker,
		})
		if resp != nil && resp.StatusCode != 200 {
			respContent, _ := ioutil.ReadAll(resp.Body)
			log.Warnf("List Multipart Uploads Response Code: %d, Response Content: %s",
				resp.StatusCode, respContent)
			return false
		} else if err != nil {
			log.Warnf(err.Error())
			return false
		}
		keyMarker = result.NextKeyMarker
		uploadIDMarker = result.NextUploadIDMarker
		isTruncated = result.IsTruncated
		for _, upload := range result.Uploads {
			partNum++
			log.Infof("Key:%s, UploadId:%s", upload.Key)
		}
	}
	log.Infof(" Parts num: %d", partNum)
	return true
}
