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
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/huanght1997/cosutil/coshelper"

	"github.com/danwakefield/fnmatch"
	"github.com/huanght1997/cos-go-sdk-v5"
	"github.com/mitchellh/go-homedir"
	"github.com/schollz/progressbar/v3"
	log "github.com/sirupsen/logrus"
)

type UploadOption struct {
	SkipMd5 bool
	Sync    bool
	Include []string
	Ignore  []string
	Force   bool
	Delete  bool
}

// PUT object can only upload 5GB file at most.
const (
	singleUploadMaxSize = 5 * 1024 * 1024 * 1024
)

var (
	pathDigest   string
	uploadID     string
	haveUploaded = make(map[int]struct{})
	md5List      []string
	uploadBar    *progressbar.ProgressBar
	uploadDone   chan bool
)

// Upload a single file.
func (client *Client) UploadFile(localPath string, cosPath string, headers *http.Header, options *UploadOption) int {
	fileSize, err := coshelper.GetFileSize(localPath)
	if err != nil {
		return 2
	}
	// Less than PartSize (MB), use put, force multipart upload if fileSize > 5GB
	if fileSize <= int64(client.Config.PartSize)*1024*1024 && fileSize <= singleUploadMaxSize {
		return client.singleUpload(localPath, cosPath, headers, options)
	}
	return client.multipartUpload(localPath, cosPath, headers, options)
}

// Upload a folder.
func (client *Client) UploadFolder(localPath string, cosPath string, headers *http.Header, options *UploadOption) int {
	successNum := 0
	skipNum := 0
	failNum := 0
	rawLocalPath, rawCosPath := localPath, cosPath
	if !strings.HasSuffix(rawLocalPath, "/") {
		rawLocalPath += "/"
	}
	if !strings.HasSuffix(rawCosPath, "/") {
		rawCosPath += "/"
	}
	// remove leading slashes
	rawCosPath = strings.TrimLeft(rawCosPath, "/")

	// q is a slice used to act as a queue
	q := make([]PathPair, 0)
	// add first element
	q = append(q, PathPair{
		LocalPath: localPath,
		CosPath:   cosPath,
	})
	uploadFileList := make([]PathPair, 0)
	// BFS upload folders
	// I can use Walk in Go, but I'd like to try multi thread upload.
	for len(q) > 0 {
		localPath = q[0].LocalPath
		cosPath = q[0].CosPath
		// remove queue head
		q = q[1:]
		// with suffix /, the path are folders
		if !strings.HasSuffix(localPath, "/") {
			localPath += "/"
		}
		if !strings.HasSuffix(cosPath, "/") {
			cosPath += "/"
		}
		// remove leading slashes
		cosPath = strings.TrimLeft(cosPath, "/")
		// Get the file list under current directory
		files, err := ioutil.ReadDir(localPath)
		if err != nil {
			log.Warn(err.Error())
			return -1
		}
		for _, file := range files {
			filePath := path.Join(localPath, file.Name())
			if file.IsDir() {
				// a subdirectory, just append it to queue to wait for the next traverse
				q = append(q, PathPair{
					LocalPath: filePath,
					CosPath:   cosPath + file.Name(),
				})
			} else {
				// a single file, add it to upload file list
				uploadFileList = append(uploadFileList, PathPair{
					LocalPath: filePath,
					CosPath:   cosPath + file.Name(),
				})
				// if 1000 files need to upload, upload them now!
				if len(uploadFileList) >= 1000 {
					succ, skip, fail := client.uploadFiles(uploadFileList, headers, options)
					successNum += succ
					skipNum += skip
					failNum += fail
					// clear upload file list
					uploadFileList = make([]PathPair, 0)
				}
			}
		}
	}
	// upload remaining upload file list
	if len(uploadFileList) > 0 {
		succ, skip, fail := client.uploadFiles(uploadFileList, headers, options)
		successNum += succ
		skipNum += skip
		failNum += fail
	}
	log.Infof("%d files uploaded, %d files skipped, %d files failed",
		successNum, skipNum, failNum)

	// if --sync and --delete flag set, delete files which not exist on COS.
	if options.Sync && options.Delete {
		if !options.Force {
			question := fmt.Sprintf("WARN: you are deleting some files in the '%s' COS path, please make sure",
				rawCosPath)
			if !coshelper.Confirm(question, "no") {
				return -3
			}
		}
		log.Info("Synchronizing delete, please wait.")
		ret, delSuccess, delFail := client.localToRemoteSyncDelete(rawLocalPath, rawCosPath)
		if ret != 0 {
			log.Warn("Sync delete fail")
		} else {
			log.Infof("%d files sync deleted, %d files sync failed",
				delSuccess, delFail)
		}
	}
	if failNum == 0 {
		return 0
	} else {
		return -1
	}
}

// upload a single file, using PUT. If upload successfully, return 0; if skipped, return -2; if failed, return -1
func (client *Client) singleUpload(localPath string, cosPath string, headers *http.Header, options *UploadOption) int {
	localMd5 := ""
	fileSize, err := coshelper.GetFileSize(localPath)
	if err != nil {
		return 2
	}
	if !options.SkipMd5 {
		// if file size > 20M
		if fileSize > 20*1024*1024 {
			log.Infof(`The MD5 of file "%s" is being calculated, please wait. If you do not need to calculate MD5, you can use --skipmd5 to skip`,
				localPath)
		}
		localMd5 = coshelper.GetFileMd5(localPath)
		log.Debugf(`The MD5 of file "%s" is "%s"`, localPath, localMd5)
	}
	if !client.localToRemoteSyncCheck(localPath, cosPath, localMd5, fileSize, options) {
		return -2
	}
	log.Infof("Upload %s   =>   cos://%s/%s",
		localPath,
		client.Config.Bucket,
		cosPath)
	for j := 0; j <= client.Config.RetryTimes; j++ {
		if j > 0 {
			log.Infof("Retry to upload %s   =>   cos://%s/%s",
				localPath, client.Config.Bucket, cosPath)
		}
		headers.Set("x-cos-meta-md5", localMd5)
		file, _ := os.Open(localPath)
		_, err := client.Client.Object.Put(context.Background(), cosPath, file, &cos.ObjectPutOptions{
			ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
				XOptionHeader: headers,
			},
		})
		_ = file.Close()
		if err != nil {
			log.Warn(err.Error())
		} else {
			return 0
		}
		if j < client.Config.RetryTimes {
			time.Sleep((1 << j) * time.Second)
		}
	}
	log.Warnf(`Upload file "%s" FAILED.`, localPath)
	return -1
}

func (client *Client) uploadFiles(uploadFileList []PathPair, headers *http.Header, options *UploadOption) (successNum, skipNum, failNum int) {
	successNum, skipNum, failNum = 0, 0, 0
	tasks := 0
	var multiUploadList []PathPair
	uploadStatus := make(chan int, client.Config.MaxThread)
	uploading := make(chan struct{}, client.Config.MaxThread)
	for _, pathPair := range uploadFileList {
		f, err := os.Stat(pathPair.LocalPath)
		if err != nil {
			failNum++
			log.Warn(err.Error())
			log.Warnf(`Upload file "%s" FAILED.`, pathPair.LocalPath)
			continue
		}
		fileSize := f.Size()
		if fileSize <= int64(client.Config.PartSize)*1024*1024 && fileSize <= singleUploadMaxSize {
			tasks++
			// start a goroutine to upload a single file.
			go func(localPath, cosPath string) {
				uploading <- struct{}{}                                                   // if max thread reached, block here
				uploadStatus <- client.singleUpload(localPath, cosPath, headers, options) // channel is a thread-safe queue
				<-uploading                                                               // upload finished, release, blocked upload goroutine will run.
			}(pathPair.LocalPath, pathPair.CosPath)
		} else {
			multiUploadList = append(multiUploadList, pathPair)
		}
	}
	for i := 0; i < tasks; i++ {
		v := <-uploadStatus // if no data, this sentence will block the main goroutine
		switch v {
		case 0:
			successNum++
		case -2:
			skipNum++
		default:
			failNum++
		}
	}
	for _, pathPair := range multiUploadList {
		ret := client.multipartUpload(pathPair.LocalPath,
			pathPair.CosPath, headers, options)
		switch ret {
		case 0:
			successNum++
		case -2:
			skipNum++
		default:
			log.Warnf(`Upload file "%s" FAILED.`, pathPair.LocalPath)
			failNum++
		}
	}
	return
}

func (client *Client) multipartUpload(localPath string, cosPath string, headers *http.Header, options *UploadOption) int {
	fileMd5 := ""
	f, err := os.Stat(localPath)
	if err != nil {
		return 2
	}
	fileSize := f.Size()
	if !options.SkipMd5 {
		log.Infof(`The MD5 of file "%s" is being calculated, please wait. If you do not need to calculate MD5, you can use --skipmd5 to skip`,
			localPath)
		fileMd5 = coshelper.GetFileMd5(localPath)
		log.Debugf(`The MD5 of file "%s" is "%s"`, localPath, fileMd5)
	}
	if !client.localToRemoteSyncCheck(localPath, cosPath, fileMd5, fileSize, options) {
		return -2
	}
	log.Infof("Upload %s   =>   cos://%s/%s",
		localPath, client.Config.Bucket, cosPath)
	headers.Set("x-cos-meta-md5", fileMd5)
	ret := client.initMultiUpload(localPath, cosPath, headers, options)
	if ret == 0 {
		log.Debug("Init multipart upload ok")
	} else {
		log.Warn("Init multipart upload failed")
		return -1
	}
	ret = client.multiUploadParts(localPath, cosPath, options)
	if ret == 0 {
		log.Debug("Multipart upload ok")
	} else {
		log.Warn("Some partial upload failed. Please retry the last command to continue")
		return -1
	}
	ret = client.completeMultiUpload(cosPath)
	if ret == 0 {
		log.Debug("Complete multipart upload ok")
	} else {
		log.Warn("Complete multipart upload failed")
		return -1
	}
	return 0
}

// Check whether this sync should be processed.
// the sync will not be processed if match one of them:
//   the file is not in include list (default include path is '*')
//   the file is in ignore list (default ignore path is empty)
//   when --sync flag specified, if the remote file and the local file has the same size and same MD5.
// if this sync should be processed, return true; if this sync should be skipped, return false.
func (client *Client) localToRemoteSyncCheck(localPath string, cosPath string, md5 string, size int64, options *UploadOption) bool {
	// check this path is in ignore or include list
	isInclude, isIgnore := false, false
	for _, rule := range options.Include {
		if fnmatch.Match(rule, cosPath, 0) {
			isInclude = true
			break
		}
	}
	for _, rule := range options.Ignore {
		if fnmatch.Match(rule, localPath, 0) {
			isIgnore = true
			break
		}
	}
	if !isInclude || isIgnore {
		log.Debugf("Skip %s", localPath)
		return false
	}
	if options.Sync {
		resp, err := client.Client.Object.Head(context.Background(), cosPath, nil)
		if err != nil {
			return true
		}
		remoteMd5 := resp.Header.Get("x-cos-meta-md5")
		remoteSize, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
		if err != nil {
			remoteSize = -1
		}
		if size == remoteSize {
			if options.SkipMd5 || strings.EqualFold(md5, remoteMd5) {
				log.Debugf("Skip %s   =>   cos://%s/%s",
					localPath, client.Config.Bucket, cosPath)
				return false
			}
		}
	}
	return true
}

// Remove objects exist on COS but not local.
func (client *Client) localToRemoteSyncDelete(localPath string, cosPath string) (ret, successNum, failNum int) {
	successNum = 0
	failNum = 0
	nextMarker := ""
	isTruncated := true
	for isTruncated {
		var deleteList []string
		for i := 0; i <= client.Config.RetryTimes; i++ {
			// get objects in the bucket
			result, _, err := client.Client.Bucket.Get(context.Background(), &cos.BucketGetOptions{
				Prefix:    cosPath,
				Delimiter: "",
				Marker:    nextMarker,
				MaxKeys:   1000,
			})
			if err != nil {
				log.Warn(err.Error())
			} else {
				// if true, some objects are not shown, continue
				isTruncated = result.IsTruncated
				nextMarker = result.NextMarker
				for _, file := range result.Contents {
					remotePath := file.Key
					localDeletePath := localPath + remotePath[len(cosPath):]
					// if there is no local file, delete the file on COS
					if !coshelper.IsFile(localDeletePath) {
						deleteList = append(deleteList, remotePath)
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

func (client *Client) initMultiUpload(localPath string, cosPath string, headers *http.Header, options *UploadOption) int {
	// If we can find unfinished task, get the UploadID.
	pathDigest = getPathDigest(localPath, cosPath)
	md5List = make([]string, 0)
	haveUploaded = make(map[int]struct{})
	if !options.Force && coshelper.IsFile(pathDigest) {
		content, err := ioutil.ReadFile(pathDigest)
		if err == nil {
			uploadID = string(content)
			if client.listPart(cosPath) {
				log.Info("Continue uploading from last breakpoint")
				return 0
			}
		}
	}
	result, resp, err := client.Client.Object.InitiateMultipartUpload(context.Background(), cosPath, &cos.InitiateMultipartUploadOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			XOptionHeader: headers,
		},
	})
	if err != nil {
		log.Warn(err.Error())
		return -1
	}
	respContent, _ := ioutil.ReadAll(resp.Body)
	log.Debugf("Init resp: %s", string(respContent))
	uploadID = result.UploadID
	tmpDir, err := homedir.Expand("~/.tmp")
	if err != nil {
		log.Warn(err.Error())
		return -1
	}
	if !coshelper.IsDir(tmpDir) {
		err := os.MkdirAll(tmpDir, os.ModePerm)
		if err != nil {
			log.Debug("Open upload tmp file error.")
		}
	}
	err = ioutil.WriteFile(pathDigest, []byte(uploadID), 0666)
	if err != nil {
		log.Debug("Open upload tmp file error.")
	}
	return 0
}

func (client *Client) multiUploadParts(localPath string, cosPath string, options *UploadOption) int {
	var offset int64 = 0
	fileSize, err := coshelper.GetFileSize(localPath)
	if err != nil {
		return -1
	}
	log.Debugf("file size: %d", fileSize)
	chunkSize := 1024 * 1024 * int64(client.Config.PartSize)
	if chunkSize >= singleUploadMaxSize {
		chunkSize = singleUploadMaxSize
	}
	// At most 10000 blocks
	for fileSize/chunkSize >= 10000 {
		chunkSize *= 10
	}
	partsNum := int(fileSize / chunkSize)
	lastSize := fileSize - int64(partsNum)*chunkSize
	haveUploadedNum := len(haveUploaded)
	if lastSize != 0 {
		partsNum++
	}
	maxThread := client.Config.MaxThread
	if maxThread > partsNum-haveUploadedNum {
		maxThread = partsNum - haveUploadedNum
	}
	uploading := make(chan struct{}, maxThread)
	uploadResult := make(chan int, maxThread)
	uploadDone = make(chan bool)
	// Initialize upload bar
	uploadBar = progressbar.NewOptions64(
		fileSize,
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowBytes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			println()
		}),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: ".",
			BarStart:      "|",
			BarEnd:        "|",
		}),
	)
	_ = uploadBar.RenderBlank()
	realPartsNum := partsNum
	for i := 0; i < partsNum; i++ {
		if _, ok := haveUploaded[i+1]; ok {
			// Just update the progress
			if partsNum == i+1 {
				go updateProgress(uploadBar, fileSize-offset, uploadDone)
				offset += fileSize - offset
			} else {
				go updateProgress(uploadBar, chunkSize, uploadDone)
				offset += chunkSize
			}
			// no need to upload
			realPartsNum--
			continue
		}
		// Upload the i-th part
		if i+1 == partsNum {
			go func(offset int64, length int64, idx int) {
				uploading <- struct{}{}
				uploadResult <- client.multiUploadPartsData(localPath, cosPath, offset, length, uploadID, idx, options)
				<-uploading
			}(offset, fileSize-offset, i+1)
		} else {
			go func(offset int64, length int64, idx int) {
				uploading <- struct{}{}
				uploadResult <- client.multiUploadPartsData(localPath, cosPath, offset, length, uploadID, idx, options)
				<-uploading
			}(offset, chunkSize, i+1)
			// update offset
			offset += chunkSize
		}
	}
	failedNum := 0
	for i := 0; i < realPartsNum; i++ {
		result := <-uploadResult
		if result != 0 {
			failedNum++
		}
	}
	select {
	case <-uploadDone:
		// Did nothing but stop blocking
		// because the progress bar will be rendered after a short sleep, which can make the output really a mess.
	case <-time.After(500 * time.Millisecond):
		// In case of something failed
		// Also do nothing.
	}
	if failedNum == 0 {
		return 0
	} else {
		return -1
	}
}

func (client *Client) multiUploadPartsData(localPath string, cosPath string, offset int64, chunkSize int64, uploadID string, index int, options *UploadOption) int {
	f, err := os.Open(localPath)
	if err != nil {
		log.Warn(err.Error())
		return -1
	}
	data := make([]byte, chunkSize)
	_, err = f.ReadAt(data, offset)
	if err != nil {
		return -1
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Warn("Close file fail")
		}
	}()
	for j := 0; j <= client.Config.RetryTimes; j++ {
		resp, err := client.Client.Object.UploadPart(context.Background(), cosPath, uploadID, index, bytes.NewReader(data), nil)
		if err != nil {
			log.Warnf("Upload part failed, key: %s, partNumber: %d, round: %d, exception: %s",
				cosPath, index, j+1, err.Error())
			time.Sleep((1 << j) * time.Second)
			continue
		}
		_, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Warnf("Upload part failed, key: %s, partNumber: %d, round: %d, exception: %s",
				cosPath, index, j+1, err.Error())
			time.Sleep((1 << j) * time.Second)
			continue
		}
		if resp.StatusCode == 200 {
			serverMd5 := resp.Header.Get("ETag")
			serverMd5 = strings.ReplaceAll(serverMd5, `"`, "")
			md5List = append(md5List, fmt.Sprintf("%d#%s", index, serverMd5))
			localEncryption := fmt.Sprintf("%x", md5.Sum(data))
			if options.SkipMd5 || serverMd5 == localEncryption {
				go updateProgress(uploadBar, chunkSize, uploadDone)
				haveUploaded[index] = struct{}{}
				return 0
			} else {
				log.Warnf("Upload part failed, key: %s, partNumber: %d, round: %d, exception: %s",
					cosPath, index, j+1, "Encryption verification is inconsistent")
				time.Sleep((1 << j) * time.Second)
				continue
			}
		}
	}
	return -1
}

func (client *Client) completeMultiUpload(cosPath string) int {
	log.Info("Completing multiupload, please wait")
	parts := make([]cos.Object, 0)
	for _, str := range md5List {
		etag := strings.Split(str, "#")[1]
		partNumber, _ := strconv.Atoi(strings.Split(str, "#")[0])
		parts = append(parts, cos.Object{
			ETag:       etag,
			PartNumber: partNumber,
		})
	}
	sort.Slice(parts, func(i, j int) bool {
		return parts[i].PartNumber < parts[j].PartNumber
	})
	completeOption := &cos.CompleteMultipartUploadOptions{
		Parts: parts,
	}
	_, resp, err := client.Client.Object.CompleteMultipartUpload(context.Background(), cosPath, uploadID, completeOption)
	if resp != nil && resp.StatusCode != 200 {
		respContent, _ := ioutil.ReadAll(resp.Body)
		log.Warnf("CompleteMultipartUpload Response Code: %d, Response Content: %s",
			resp.StatusCode, string(respContent))
		return -1
	} else if err != nil {
		log.Warn(err.Error())
		return -1
	}
	err = os.Remove(pathDigest)
	if err != nil {
		log.Warnf("Delete temporary digest file '%s' failed, please delete it manually", pathDigest)
	}
	return 0
}

func (client *Client) listPart(cosPath string) bool {
	log.Debug("getting uploaded parts")
	nextMarker := ""
	isTruncated := true
	for isTruncated {
		result, resp, err := client.Client.Object.ListParts(context.Background(), cosPath, uploadID, &cos.ObjectListPartsOptions{
			MaxParts:         "1000",
			PartNumberMarker: nextMarker,
		})
		if err != nil {
			return false
		}
		if resp.StatusCode == 200 {
			isTruncated = result.IsTruncated
			nextMarker = result.NextPartNumberMarker
			content, _ := ioutil.ReadAll(resp.Body)
			log.Debugf("list resp, status code: %d, headers: %v, text: %s",
				resp.StatusCode, resp.Header, string(content))
			for _, content := range result.Parts {
				id := content.PartNumber
				haveUploaded[id] = struct{}{}
				md5List = append(md5List, fmt.Sprintf("%d#%s", id, strings.ReplaceAll(content.ETag, `"`, "")))
			}
		} else {
			content, _ := ioutil.ReadAll(resp.Body)
			log.Warnf("ListParts Response Code: %d, Response Content: %s", resp.StatusCode, string(content))
			return false
		}
	}
	return true
}

func getPathDigest(localPath string, cosPath string) string {
	localAbsPath, err := filepath.Abs(localPath)
	if err != nil {
		panic(err)
	}
	fileSize, err := coshelper.GetFileSize(localPath)
	if err != nil {
		panic(err)
	}
	ori := fmt.Sprintf("%s!!!%d!!!%s", localAbsPath, fileSize, cosPath)
	md5sum := fmt.Sprintf("%x", md5.Sum([]byte(ori)))
	file, err := homedir.Expand("~/.tmp/" + md5sum)
	if err != nil {
		panic(err)
	}
	return file
}
