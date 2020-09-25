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

type AbortFile struct {
	Key      string
	UploadID string
}

func (client *Client) AbortParts(cosPath string) int {
	nextKeyMarker := ""
	nextUploadIDMarker := ""
	isTruncated := true
	successNum, failNum := 0, 0
	for isTruncated {
		abortList := make([]AbortFile, 0)
		for i := 0; i < client.Config.RetryTimes; i++ {
			result, resp, err := client.Client.Bucket.ListMultipartUploads(context.Background(),
				&cos.ListMultipartUploadsOptions{
					Prefix:         cosPath,
					MaxUploads:     1000,
					KeyMarker:      nextKeyMarker,
					UploadIDMarker: nextUploadIDMarker,
				})
			if err != nil {
				log.Warn(err.Error())
			} else if resp.StatusCode != 200 {
				respContent, _ := ioutil.ReadAll(resp.Body)
				log.Warnf("Response Code: %d, Response: %s",
					resp.StatusCode, respContent)
			} else {
				isTruncated = result.IsTruncated
				nextUploadIDMarker = result.NextUploadIDMarker
				nextKeyMarker = result.NextKeyMarker
				for _, file := range result.Uploads {
					abortList = append(abortList, AbortFile{
						Key:      file.Key,
						UploadID: uploadID,
					})
				}
				for _, file := range abortList {
					resp, err := client.Client.Object.AbortMultipartUpload(context.Background(),
						file.Key, file.UploadID)
					if resp != nil && resp.StatusCode != 200 {
						respContent, _ := ioutil.ReadAll(resp.Body)
						log.Warnf("Response Code: %d, Response: %s",
							resp.StatusCode, respContent)
						log.Infof("Abort key: %s, UploadId: %s failed",
							file.Key, file.UploadID)
						failNum++
					} else if err != nil {
						log.Warnf(err.Error())
						log.Infof("Abort key: %s, UploadId: %s failed",
							file.Key, file.UploadID)
						failNum++
					} else {
						log.Infof("Abort key: %s, UploadId: %s",
							file.Key, file.UploadID)
						successNum++
					}
				}
				break
			}
		}
	}
	log.Infof("%d files successful, %d files failed",
		successNum, failNum)
	if failNum != 0 {
		return -1
	}
	return 0
}
