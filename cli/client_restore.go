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
	"time"

	"github.com/huanght1997/cos-go-sdk-v5"
	log "github.com/sirupsen/logrus"
)

const (
	Expedited = iota
	Standard
	Bulk
)

type RestoreOption struct {
	Day  int
	Tier int
}

func (client *Client) RestoreFolder(cosPath string, options *RestoreOption) int {
	successNum, progressNum, failNum := 0, 0, 0
	nextMarker := ""
	isTruncated := true
	restoring := make(chan struct{}, client.Config.MaxThread)
	restoreResult := make(chan int, client.Config.MaxThread)
	for isTruncated {
		for i := 0; i <= client.Config.RetryTimes; i++ {
			result, resp, err := client.Client.Bucket.Get(context.Background(), &cos.BucketGetOptions{
				Prefix:  cosPath,
				Marker:  nextMarker,
				MaxKeys: 1000,
			})
			if resp != nil && resp.StatusCode != 200 {
				respContent, _ := ioutil.ReadAll(resp.Body)
				log.Warnf("Bucket Get Response Code: %d, Response Content: %s",
					resp.StatusCode, string(respContent))
			} else if err != nil {
				log.Warn(err.Error())
			} else {
				isTruncated = result.IsTruncated
				nextMarker = result.NextMarker
				for _, file := range result.Contents {
					go func(path string) {
						restoring <- struct{}{}
						restoreResult <- client.RestoreFile(path, options)
						<-restoring
					}(file.Key)
				}
				for j := 0; j < len(result.Contents); j++ {
					v := <-restoreResult
					switch v {
					case 0:
						successNum++
					case -2:
						progressNum++
					default:
						failNum++
					}
				}
				break
			}
			if i == client.Config.RetryTimes {
				return -1
			} else {
				time.Sleep((1 << i) * time.Second)
			}
		}
	}
	log.Infof("%d files successful, %d files have in progress, %d files failed",
		successNum, progressNum, failNum)
	if failNum == 0 {
		return 0
	} else {
		return -1
	}
}

func (client *Client) RestoreFile(cosPath string, options *RestoreOption) int {
	tier := ""
	switch options.Tier {
	case Expedited:
		tier = "Expedited"
	case Standard:
		tier = "Standard"
	case Bulk:
		tier = "Bulk"
	}
	log.Infof("Restore cos://%s/%s", client.Config.Bucket, cosPath)
	resp, err := client.Client.Object.PostRestore(context.Background(), cosPath, &cos.ObjectRestoreOptions{
		Days: options.Day,
		Tier: &cos.CASJobParameters{Tier: tier},
	})
	if resp != nil {
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return 0
		} else {
			respContent, _ := ioutil.ReadAll(resp.Body)
			if resp.StatusCode == 409 && strings.Contains(string(respContent), "RestoreAlreadyInProgress") {
				log.Warnf("cos://%s/%s already in progress",
					client.Config.Bucket, cosPath)
				return -2
			} else {
				log.Warnf("Post Restore Response Code: %d, Response Content: %s",
					resp.StatusCode, string(respContent))
				return -1
			}
		}
	} else if err != nil {
		log.Warnf(err.Error())
		return -1
	}
	return -1
}
