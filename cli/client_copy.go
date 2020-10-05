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
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/huanght1997/cosutil/coshelper"

	"github.com/danwakefield/fnmatch"
	log "github.com/sirupsen/logrus"
	"github.com/tencentyun/cos-go-sdk-v5"
)

type CopyOption struct {
	Sync      bool
	Force     bool
	Directive string
	SkipMd5   bool
	Ignore    []string
	Include   []string
	Delete    bool
	Move      bool
}

// sourcePath: bucket-appid.cos.ap-guangzhou.myqcloud.com/path/
// cosPath: test/
func (client *Client) CopyFolder(sourcePath string, cosPath string, headers *http.Header, options *CopyOption) int {
	if !strings.HasSuffix(cosPath, "/") {
		cosPath += "/"
	}
	if !strings.HasSuffix(sourcePath, "/") {
		sourcePath += "/"
	}
	cosPath = strings.TrimLeft(cosPath, "/")
	successNum, skipNum, failNum := 0, 0, 0
	nextMarker := ""
	isTruncated := true
	sourceClient, err := client.sourcePathToClient(sourcePath)
	if err != nil {
		return -1
	}
	// sourceSchema: bucket-appid.cos.ap-guangzhou.myqcloud.com/
	sourceSchema := strings.Split(sourcePath, "/")[0] + "/"
	sourcePath = sourcePath[len(sourceSchema):]
	// sourcePath: path/
	rawSourcePath := sourcePath
	rawCosPath := cosPath
	if !strings.HasSuffix(rawSourcePath, "/") {
		rawSourcePath += "/"
	}
	if !strings.HasSuffix(rawCosPath, "/") {
		rawCosPath += "/"
	}
	rawCosPath = strings.TrimLeft(rawCosPath, "/")
	copying := make(chan struct{}, client.Config.MaxThread)
	copyResults := make(chan int, client.Config.MaxThread)
	task := 0
	for isTruncated {
		var i int
		for i = 0; i <= client.Config.RetryTimes; i++ {
			result, _, err := sourceClient.Client.Bucket.Get(context.Background(), &cos.BucketGetOptions{
				Prefix:    sourcePath,
				Delimiter: "",
				Marker:    nextMarker,
				MaxKeys:   1000,
			})
			if err != nil {
				log.Warn(err.Error())
			} else {
				isTruncated = result.IsTruncated
				nextMarker = result.NextMarker
				for _, file := range result.Contents {
					filePath := file.Key
					fileSourcePath := sourceSchema + filePath
					var fileCosPath string
					if !strings.HasSuffix(sourcePath, "/") && len(sourcePath) != 0 {
						// sourcePath is a folder!
						fileCosPath = cosPath + filePath[len(sourcePath)+1:]
					} else {
						fileCosPath = cosPath + filePath[len(sourcePath):]
					}
					task++
					go func(sourcePath, cosPath string) {
						copying <- struct{}{}
						copyResults <- client.CopyFile(sourcePath, cosPath, headers, options)
						<-copying
					}(fileSourcePath, fileCosPath)
				}
				break
			}
			// if it is the last time, do not sleep again.
			if i < client.Config.RetryTimes {
				time.Sleep((1 << i) * time.Second)
			}
		}
		if i > client.Config.RetryTimes {
			log.Warn("ListObjects fail")
			return -1
		}
	}
	for i := 0; i < task; i++ {
		v := <-copyResults
		switch v {
		case 0:
			successNum++
		case -2:
			skipNum++
		default:
			failNum++
		}
	}
	if options.Move {
		log.Infof("%d files moved, %d files skipped, %d files failed",
			successNum, skipNum, failNum)
	} else {
		log.Infof("%d files copied, %d files skipped, %d files failed",
			successNum, skipNum, failNum)
	}
	if options.Sync && options.Delete {
		if !options.Force {
			if !coshelper.Confirm("WARN: you are deleting some files in the '%s' COS path, please make sure", "no") {
				return -3
			}
		}
		log.Info("Synchronizing delete, please wait.")
		ret, delSucc, delFail := client.remoteToRemoteSyncDelete(sourceClient, rawSourcePath, rawCosPath)
		if ret != 0 {
			log.Warn("Sync delete fail")
		} else {
			log.Infof("%d files sync deleted, %d files sync failed",
				delSucc, delFail)
		}
	}
	if failNum != 0 {
		return -1
	}
	return 0
}

// sourcePath: bucket-appid.cos.ap-guangzhou.myqcloud.com/path/to/file
// cosPath: test/file
func (client *Client) CopyFile(sourcePath string, cosPath string, headers *http.Header, options *CopyOption) int {
	sourceClient, err := client.sourcePathToClient(sourcePath)
	if err != nil {
		return -1
	}
	if !client.remoteToRemoteSyncCheck(sourcePath, cosPath, options) {
		return -2
	}
	if options.Move {
		log.Info("Move cos://%s/%s   =>   cos://%s/%s",
			sourceClient.Config.Bucket, sourcePath[strings.Index(sourcePath, "/")+1:],
			client.Config.Bucket, cosPath)
	}
	// Check whether a single Copy interface could be use.
	justCopy := false
	// if less than 5GB, just use it.
	// if the source and the target COS bucket are in the same region, just use it.
	resp, err := sourceClient.Client.Object.Head(context.Background(), sourcePath[strings.Index(sourcePath, "/")+1:], nil)
	if err != nil {
		log.Warn(err.Error())
		return -1
	}
	fileSize, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if fileSize < singleUploadMaxSize {
		justCopy = true
	}
	if sourceClient.Config.Endpoint == client.Config.Endpoint &&
		resp.Header.Get("x-cos-storage-class") == "" {
		// we now only support copy to another bucket with STANDARD storage class.
		justCopy = true
	}
	if justCopy {
		_, _, err = client.Client.Object.Copy(context.Background(), cosPath, sourcePath, nil)
		if err != nil {
			log.Warn(err.Error())
			return -1
		}
	} else {
		// Create Multipart upload first.
		result, _, err := client.Client.Object.InitiateMultipartUpload(context.Background(), cosPath, &cos.InitiateMultipartUploadOptions{
			ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
				XOptionHeader: headers,
			},
		})
		if err != nil {
			log.Warn(err.Error())
			return -1
		}
		uploadID = result.UploadID
		// Do multipart upload (copy).
		chunkSize := 1024 * 1024 * int64(client.Config.PartSize)
		if chunkSize >= singleUploadMaxSize {
			chunkSize = singleUploadMaxSize
		}
		partsNum := int(fileSize / chunkSize)
		lastSize := fileSize - int64(partsNum)*chunkSize
		if lastSize != 0 {
			partsNum++
		}
		copying := make(chan struct{}, client.Config.MaxThread)
		copyResult := make(chan string, client.Config.MaxThread)
		for i := 0; i < partsNum; i++ {
			startOffset := int64(i) * chunkSize
			endOffset := startOffset + chunkSize - 1
			if i == partsNum-1 {
				endOffset = fileSize - 1
			}
			go func(idx int, start, end int64) {
				copying <- struct{}{}
				for j := 0; j <= client.Config.RetryTimes; j++ {
					result, _, err := client.Client.Object.CopyPart(context.Background(), cosPath, uploadID, idx, sourcePath, &cos.ObjectCopyPartOptions{
						XCosCopySourceRange: fmt.Sprintf("bytes=%d-%d", start, end),
					})
					if err != nil {
						log.Warnf("An error occurred when copying the %d part (total %d), retry time: %d, error message: '%s'",
							idx, partsNum, j, err.Error())
						// retry
						if j == client.Config.RetryTimes {
							copyResult <- fmt.Sprintf("%d#%s", idx, "error")
							break
						}
						time.Sleep((1 << j) * time.Second)
					} else {
						copyResult <- fmt.Sprintf("%d#%s", idx, result.ETag)
						break
					}
				}
			}(i+1, startOffset, endOffset)
			<-copying
		}
		// Complete multipart upload.
		parts := make([]cos.Object, 0)
		failed := false
		for i := 0; i < partsNum; i++ {
			v := <-copyResult
			etag := strings.Split(v, "#")[1]
			if etag == "error" {
				failed = true
			}
			partNumber, _ := strconv.Atoi(strings.Split(v, "#")[0])
			parts = append(parts, cos.Object{
				ETag:       etag,
				PartNumber: partNumber,
			})
		}
		if failed {
			log.Warn("Failed to copy some parts.")
			client.AbortParts(cosPath)
			return -1
		}
		sort.Slice(parts, func(i, j int) bool {
			return parts[i].PartNumber < parts[j].PartNumber
		})
		completeOption := &cos.CompleteMultipartUploadOptions{
			Parts: parts,
		}
		_, resp, err = client.Client.Object.CompleteMultipartUpload(context.Background(), cosPath, uploadID, completeOption)
		if err != nil {
			log.Warn(err.Error())
			return -1
		}
	}
	if options.Move {
		sourceClient.DeleteFile(sourcePath[strings.Index(sourcePath, "/")+1:], &DeleteOption{
			Force:    true,
			Versions: false,
		})
	}
	return 0
}

// Delete objects source client does not have but target client has.
func (client *Client) remoteToRemoteSyncDelete(sourceClient *Client, sourcePath string, cosPath string) (ret, successNum, failNum int) {
	successNum = 0
	failNum = 0
	nextMarker := ""
	isTruncated := true
	for isTruncated {
		var deleteList []string
		for i := 0; i <= client.Config.RetryTimes; i++ {
			// get objects in the bucket
			result, resp, err := client.Client.Bucket.Get(context.Background(), &cos.BucketGetOptions{
				Prefix:    cosPath,
				Delimiter: "",
				Marker:    nextMarker,
				MaxKeys:   1000,
			})
			if resp != nil && resp.StatusCode != 200 {
				respContent, _ := ioutil.ReadAll(resp.Body)
				log.Warnf("Bucket Get Response Code: %d, Response Content: %s", resp.StatusCode, respContent)
			} else if err != nil {
				log.Warn(err.Error())
			} else {
				// if true, some objects are not shown, continue
				isTruncated = result.IsTruncated
				nextMarker = result.NextMarker
				for _, file := range result.Contents {
					fileCosPath := file.Key
					fileSourcePath := sourcePath + fileCosPath[len(cosPath):]
					// if there is no file in source client, add it to deleteList
					resp, _ := sourceClient.Client.Object.Head(context.Background(), fileSourcePath, nil)
					if resp != nil && resp.StatusCode == 404 {
						deleteList = append(deleteList, fileCosPath)
					}
				}
				// no more retry
				break
			}
			if i >= client.Config.RetryTimes {
				return -1, successNum, failNum
			}
			time.Sleep((1 << i) * time.Second)
		}
		succ, fail := client.DeleteObjects(deleteList)
		successNum += succ
		failNum += fail
	}
	return 0, successNum, failNum
}

// sourcePath to Client
func (client *Client) sourcePathToClient(sourcePath string) (*Client, error) {
	sourceTmpPath := strings.Split(sourcePath, "/")
	sourceTmpPath = strings.Split(sourceTmpPath[0], ".")
	if len(sourceTmpPath) < 2 {
		return nil, coshelper.Error{
			Code:    1,
			Message: "Invalid source path",
		}
	}
	// sourceBucket: bucket-appid
	sourceBucket := sourceTmpPath[0]
	// sourceEndpoint: ap-guangzhou.myqcloud.com
	sourceEndpoint := strings.Join(sourceTmpPath[1:], ".")
	sourceConfig := *client.Config // copy a config from client, dereference it.
	sourceConfig.Endpoint = sourceEndpoint
	sourceConfig.Bucket = sourceBucket
	return NewClient(&sourceConfig), nil
}

func (client *Client) remoteToRemoteSyncCheck(sourcePath, cosPath string, options *CopyOption) bool {
	sourceKey := sourcePath[strings.Index(sourcePath, "/")+1:]
	// check this path is in ignore or include list
	isInclude, isIgnore := false, false
	for _, rule := range options.Include {
		if fnmatch.Match(rule, sourceKey, 0) {
			isInclude = true
			break
		}
	}
	for _, rule := range options.Ignore {
		if fnmatch.Match(rule, sourceKey, 0) {
			isIgnore = true
			break
		}
	}
	sourceClient, err := client.sourcePathToClient(sourcePath)
	if err != nil {
		log.Warn(err.Error())
		return true
	}
	if !isInclude || isIgnore {
		log.Debugf("Skip cos://%s/%s => cos://%s/%s",
			sourceClient.Config.Bucket, sourceKey,
			client.Config.Bucket, cosPath)
		return false
	}
	if !options.Force && options.Sync {
		srcMd5, dstMd5 := "src", "dst"
		var srcSize, dstSize int64 = -1, -2
		sourceResp, err := sourceClient.Client.Object.Head(context.Background(), sourceKey, nil)
		if err != nil {
			return true
		} else if sourceResp.StatusCode == 200 {
			srcMd5 = sourceResp.Header.Get("x-cos-meta-md5")
			srcSize = sourceResp.ContentLength
		}
		targetResp, err := client.Client.Object.Head(context.Background(), cosPath, nil)
		if err != nil {
			return true
		} else if targetResp.StatusCode == 200 {
			dstMd5 = targetResp.Header.Get("x-cos-meta-md5")
			dstSize = targetResp.ContentLength
		}
		if (options.SkipMd5 || srcMd5 == dstMd5) && dstSize == srcSize {
			log.Debugf("Skip cos://%s/%s => cos://%s/%s",
				sourceClient.Config.Bucket, sourceKey,
				client.Config.Bucket, cosPath)
			return false
		}
	}
	return true
}
