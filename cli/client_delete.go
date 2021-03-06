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
	"time"

	"github.com/huanght1997/cosutil/coshelper"

	log "github.com/sirupsen/logrus"
	"github.com/tencentyun/cos-go-sdk-v5"
)

type DeleteOption struct {
	Force     bool
	Yes       bool
	Versions  bool
	VersionID string
}

func (client *Client) DeleteFolder(cosPath string, options *DeleteOption) int {
	if !options.Force && !options.Yes {
		if !coshelper.Confirm(fmt.Sprintf("WARN: you are deleting the file in the %s COS path, please make sure", cosPath), "no") {
			return -3
		}
	}
	versions := options.Versions
	if cosPath == "/" {
		cosPath = ""
	}
	haveDeletedNum := 0
	totalDeleteFileNum := 0
	nextMarker := ""
	keyMarker := ""
	versionIDMarker := ""
	isTruncated := true
	for isTruncated {
		deleteList := make([]cos.Object, 0)
		var result interface{}
		for i := 0; i <= client.Config.RetryTimes; i++ {
			var err error
			if versionIDMarker == "null" {
				versionIDMarker = ""
			}
			if versions {
				result, _, err = client.Client.Bucket.GetObjectVersions(context.Background(), &cos.BucketGetObjectVersionsOptions{
					Prefix:          cosPath,
					KeyMarker:       keyMarker,
					VersionIdMarker: versionIDMarker,
					MaxKeys:         1000,
				})
			} else {
				result, _, err = client.Client.Bucket.Get(context.Background(), &cos.BucketGetOptions{
					Prefix:  cosPath,
					Marker:  nextMarker,
					MaxKeys: 1000,
				})
			}
			if err != nil {
				log.Warn(err.Error())
			} else {
				break
			}
			if i >= client.Config.RetryTimes {
				return -1
			}
			time.Sleep((1 << i) * time.Second)
		}
		if versions {
			rt := result.(*cos.BucketGetObjectVersionsResult)
			isTruncated = rt.IsTruncated
			keyMarker = rt.NextKeyMarker
			versionIDMarker = rt.NextVersionIdMarker
			// if delete marker found, this version is specified for deleting.
			for _, file := range rt.DeleteMarker {
				deleteList = append(deleteList, cos.Object{
					Key:       file.Key,
					VersionId: file.VersionId,
				})
			}
			// History file
			for _, file := range rt.Version {
				deleteList = append(deleteList, cos.Object{
					Key:       file.Key,
					VersionId: file.VersionId,
				})
			}
		} else {
			rt := result.(*cos.BucketGetResult)
			isTruncated = rt.IsTruncated
			nextMarker = rt.NextMarker
			for _, file := range rt.Contents {
				deleteList = append(deleteList, cos.Object{
					Key: file.Key,
				})
			}
			totalDeleteFileNum += len(rt.Contents)
		}
		if len(deleteList) > 0 {
			delResult, resp, err := client.Client.Object.DeleteMulti(context.Background(), &cos.ObjectDeleteMultiOptions{
				Objects: deleteList,
			})
			if err == nil && resp.StatusCode == 200 {
				for _, file := range delResult.DeletedObjects {
					if versions {
						log.Infof("Delete %s, versionId: %s", file.Key, file.VersionId)
					} else {
						log.Infof("Delete %s", file.Key)
					}
				}
				haveDeletedNum += len(delResult.DeletedObjects)
				for _, file := range delResult.Errors {
					if versions {
						log.Infof("Delete %s, versionId: %s fail, code: %s, msg: %s",
							file.Key, file.VersionId, file.Code, file.Message)
					} else {
						log.Infof("Delete %s fail, code: %s, msg: %s", file.Key, file.Code, file.Message)
					}
				}
				if versions {
					totalDeleteFileNum += len(delResult.DeletedObjects) + len(delResult.Errors)
				}
			}
		}
	}
	if totalDeleteFileNum == 0 {
		log.Infof("The directory does not exist")
		return -1
	}
	if !versions {
		log.Infof("%d files successful, %d files failed", haveDeletedNum, totalDeleteFileNum-haveDeletedNum)
	}
	if totalDeleteFileNum == haveDeletedNum {
		return 0
	}
	return -1
}

func (client *Client) DeleteFile(cosPath string, options *DeleteOption) int {
	if !options.Force && !options.Yes {
		if !coshelper.Confirm(fmt.Sprintf("WARN: you are deleting the file in the %s COS path, please make sure", cosPath), "no") {
			return -3
		}
	}
	resp, err := client.Client.Object.Delete(context.Background(), cosPath, &cos.ObjectDeleteOptions{
		VersionId: options.VersionID,
	})
	if err != nil {
		log.Warn(err.Error())
		return -1
	}
	if resp != nil {
		respContent, _ := ioutil.ReadAll(resp.Body)
		log.Debugf("Delete Response Code: %d, Headers: %v, Response Content: %s",
			resp.StatusCode, resp.Header, string(respContent))
	}
	if resp.StatusCode == 204 || resp.StatusCode == 200 {
		if options.VersionID == "" {
			log.Infof("Delete cos://%s/%s",
				client.Config.Bucket, cosPath)
		} else {
			log.Infof("Delete cos://%s/%s?versionId=%s",
				client.Config.Bucket, cosPath, options.VersionID)
		}
		return 0
	} else {
		return -1
	}
}

func (client *Client) DeleteObjects(deleteList []string) (successNum int, failNum int) {
	successNum, failNum = 0, 0
	if len(deleteList) > 0 {
		objects := make([]cos.Object, len(deleteList))
		for i, key := range deleteList {
			objects[i].Key = key
		}
		options := &cos.ObjectDeleteMultiOptions{
			Objects: objects,
		}
		result, _, err := client.Client.Object.DeleteMulti(context.Background(), options)
		if err != nil {
			log.Warn(err)
			return 0, len(deleteList)
		}
		for _, file := range result.DeletedObjects {
			log.Infof("Delete cos://%s/%s", client.Config.Bucket, file.Key)
		}
		successNum += len(result.DeletedObjects)
		for _, file := range result.Errors {
			log.Infof("Delete cos://%s/%s fail, code: %s, msg: %s",
				client.Config.Bucket, file.Key, file.Code, file.Message)
		}
		failNum += len(result.Errors)
	}
	return
}
