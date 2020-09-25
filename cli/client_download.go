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
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
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

type DownloadOption struct {
	Force   bool
	Sync    bool
	Num     int
	Ignore  []string
	Include []string
	SkipMd5 bool
	Delete  bool
}

type multiDownloadFile struct {
	cosPath, localPath string
	size               int64
}

const (
	multiDownloadThreshold = 20 * 1024 * 1024
)

var (
	downloadBar  *progressbar.ProgressBar
	downloadDone chan bool
)

func (client *Client) DownloadFolder(cosPath string, localPath string, options *DownloadOption) int {
	// Make cosPath and localPath folder-like string
	if !strings.HasSuffix(cosPath, "/") {
		cosPath += "/"
	}
	if !strings.HasSuffix(localPath, "/") {
		localPath += "/"
	}
	cosPath = strings.TrimLeft(cosPath, "/")
	nextMarker := ""
	isTruncated := true
	successNum, failNum, skipNum := 0, 0, 0

	for isTruncated {
		downloading := make(chan int, client.Config.MaxThread)
		downloadResult := make(chan int, client.Config.MaxThread)
		multiDownloadFileList := make([]multiDownloadFile, 0)
		result, resp, err := client.Client.Bucket.Get(context.Background(), &cos.BucketGetOptions{
			Prefix:  cosPath,
			Marker:  nextMarker,
			MaxKeys: 1000,
		})
		if resp != nil && resp.StatusCode != 200 {
			respContent, _ := ioutil.ReadAll(resp.Body)
			log.Warnf("Bucket Get Response Code: %d, Response Content: %s", resp.StatusCode, string(respContent))
			log.Warn("List object failed")
			return -1
		} else if err != nil {
			log.Warn(err.Error())
			log.Warn("List object failed")
			return -1
		}
		isTruncated = result.IsTruncated
		nextMarker = result.NextMarker
		tasks := 0
		for _, file := range result.Contents {
			fileCosPath := file.Key
			fileSize := file.Size
			fileLocalPath := localPath + fileCosPath[len(cosPath):]
			// if fileCosPath has suffix /, it is an empty folder, ignore it.
			if strings.HasSuffix(fileCosPath, "/") {
				continue
			}
			if fileSize <= multiDownloadThreshold {
				// small file, just download it now.
				tasks++
				go func(cosPath, localPath string) {
					downloading <- 1
					downloadResult <- client.singleDownload(cosPath, localPath, options)
					<-downloading
				}(fileCosPath, fileLocalPath)
			} else {
				// large file, download later.
				multiDownloadFileList = append(multiDownloadFileList, multiDownloadFile{
					cosPath:   fileCosPath,
					localPath: fileLocalPath,
					size:      int64(fileSize),
				})
			}
		}
		// Stat small file.
		for i := 0; i < tasks; i++ {
			v := <-downloadResult
			switch v {
			case 0:
				successNum++
			case -2:
				skipNum++
			default:
				failNum++
			}
		}
		// Download large file one by one.
		for _, f := range multiDownloadFileList {
			ret := client.multipartDownload(f.cosPath, f.localPath, f.size, options)
			switch ret {
			case 0:
				successNum++
			case -2:
				skipNum++
			default:
				failNum++
			}
		}
	}
	log.Infof("%d files downloaded, %d files skipped, %d files failed",
		successNum, skipNum, failNum)
	// --sync --delete to delete files not in COS but in local
	if options.Sync && options.Delete {
		if !options.Force {
			if !coshelper.Confirm(fmt.Sprintf("WARN: you are deleting the file in the '%s' local path, please make sure", localPath), "no") {
				return -3
			}
		}
		log.Info("Synchronizing delete, please wait.")
		ret, delSucc, delFail := client.remoteToLocalSyncDelete(localPath, cosPath)
		if ret != 0 {
			log.Warn("sync delete fail")
		} else {
			log.Infof("%d files sync deleted, %d files sync failed",
				delSucc, delFail)
		}
	}
	if failNum == 0 {
		return 0
	} else {
		return -1
	}
}

func (client *Client) DownloadFile(cosPath string, localPath string, _ *http.Header, options *DownloadOption) int {
	resp, err := client.Client.Object.Head(context.Background(), cosPath, nil)
	if err != nil {
		log.Warn(err.Error())
		return -1
	}
	if resp.StatusCode != 200 {
		log.Warnf("Object HEAD Response Code: %d", resp.StatusCode)
		return -1
	}
	absLocalPath, err := homedir.Expand(localPath)
	if err != nil {
		log.Warn(err.Error())
		return -1
	}
	fileSize, _ := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if fileSize <= multiDownloadThreshold || options.Num == 1 {
		return client.singleDownload(cosPath, absLocalPath, options)
	} else {
		return client.multipartDownload(cosPath, absLocalPath, fileSize, options)
	}
}

func (client *Client) singleDownload(cosPath string, localPath string, options *DownloadOption) int {
	for strings.HasPrefix(cosPath, "/") {
		cosPath = cosPath[1:]
	}
	ret := client.remoteToLocalSyncCheck(cosPath, localPath, options)
	if ret != 0 {
		return ret
	}
	log.Infof("Download cos://%s/%s   =>   %s",
		client.Config.Bucket, cosPath, localPath)
	resp, err := client.Client.Object.Get(context.Background(), cosPath, nil)
	if err != nil {
		log.Warn(err.Error())
		return -1
	}
	if resp.StatusCode != 200 {
		respContent, _ := ioutil.ReadAll(resp.Body)
		log.Warnf("Object GET Response Code: %d, Content: %s", resp.StatusCode, string(respContent))
		return -1
	}
	dirPath := filepath.Dir(localPath)
	// create directories for downloaded file
	if !coshelper.IsDir(dirPath) {
		err := os.MkdirAll(dirPath, 0755)
		if err != nil {
			log.Warnf("Cannot create directory '%s'", dirPath)
		}
	}
	f, err := os.Create(localPath)
	if err != nil {
		return -1
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Warn("Cannot close file")
		}
	}()
	// make a buffer to keep chunks (1M)
	buf := make([]byte, 1024*1024)
	for {
		n, err := resp.Body.Read(buf)
		// if there is an error and not EOF, something wrong.
		if err != nil && err != io.EOF {
			return -1
		}
		// if nothing read, the file is completely read.
		if n == 0 {
			break
		}

		// Write the n bytes read to file.
		if _, err := f.Write(buf[:n]); err != nil {
			return -1
		}
	}
	return 0
}

func (client *Client) multipartDownload(cosPath string, localPath string, fileSize int64, options *DownloadOption) int {
	cosPath = strings.TrimLeft(cosPath, "/")
	ret := client.remoteToLocalSyncCheck(cosPath, localPath, options)
	if ret != 0 {
		return ret
	}
	log.Infof("Download cos://%s/%s   =>   %s",
		client.Config.Bucket, cosPath, localPath)
	var offset int64 = 0
	partsNum := options.Num
	chuckSize := fileSize / int64(partsNum)
	lastSize := fileSize - int64(partsNum)*chuckSize
	haveDownloadedNum := 0
	if lastSize != 0 {
		partsNum += 1
	}
	maxThread := client.Config.MaxThread
	if maxThread > partsNum-haveDownloadedNum {
		maxThread = partsNum - haveDownloadedNum
	}
	downloading := make(chan struct{}, maxThread)
	downloadResult := make(chan int, maxThread)
	downloadDone = make(chan bool)
	log.Debugf("chuck_size: %d", chuckSize)
	log.Debug("download file concurrently")
	log.Infof("Downloading %s", localPath)

	dirPath := filepath.Dir(localPath)
	if !coshelper.IsDir(dirPath) {
		err := os.MkdirAll(dirPath, 0755)
		if err != nil {
			log.Warnf("Cannot create directory '%s'", dirPath)
		}
	}
	// Create an empty file
	// the file must have been created when use f.Seek()
	f, err := os.Create(localPath)
	if err != nil {
		log.Warn(err.Error())
		return -1
	}
	err = f.Close()
	if err != nil {
		log.Warn(err.Error())
	}

	downloadBar = progressbar.NewOptions64(
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
	_ = downloadBar.RenderBlank()
	for i := 0; i < partsNum; i++ {
		if i+1 == partsNum {
			go func(offset, length int64, index int) {
				downloading <- struct{}{}
				downloadResult <- client.getPartsData(localPath, cosPath, offset, length)
				<-downloading
			}(offset, fileSize-offset, i+1)
		} else {
			go func(offset, length int64, index int) {
				downloading <- struct{}{}
				downloadResult <- client.getPartsData(localPath, cosPath, offset, length)
				<-downloading
			}(offset, chuckSize, i+1)
			offset += chuckSize
		}
	}
	failNum := 0
	for i := 0; i < partsNum; i++ {
		v := <-downloadResult
		if v != 0 {
			failNum++
		}
	}
	if failNum > 0 {
		log.Infof("%d parts download failed", failNum)
		err = os.Remove(localPath)
		if err != nil {
			log.Warn("delete temporary file failed.")
		}
		return -1
	}
	select {
	case <-downloadDone:
		// Did nothing but stop blocking
	case <-time.After(500 * time.Millisecond):
		// In case of something wrong
	}
	return 0
}

func (client *Client) getPartsData(localPath string, cosPath string, offset int64, length int64) int {
	for j := 0; j <= client.Config.RetryTimes; j++ {
		resp, err := client.Client.Object.Get(context.Background(), cosPath, &cos.ObjectGetOptions{
			Range: fmt.Sprintf("bytes=%d-%d",
				offset, offset+length-1),
		})
		f, err := os.OpenFile(localPath, os.O_RDWR, 0644)
		if err != nil {
			log.Warn(err.Error())
			time.Sleep((1 << j) * time.Second)
			continue
		}
		_, err = f.Seek(offset, 0)
		if err != nil {
			log.Warn(err.Error())
			time.Sleep((1 << j) * time.Second)
			continue
		}
		// make a buffer to keep chunks
		buf := make([]byte, 1024*1024)
		for {
			n, err := resp.Body.Read(buf)
			if err != nil && err != io.EOF {
				if ferr := f.Close(); ferr != nil {
					log.Warn(ferr.Error())
				}
				time.Sleep((1 << j) * time.Second)
				log.Warn(err.Error())
				continue
			}
			if n == 0 {
				break
			}

			if _, err := f.Write(buf[:n]); err != nil {
				time.Sleep((1 << j) * time.Second)
				log.Warn(err.Error())
				continue
			}
			go updateProgress(downloadBar, int64(n), downloadDone)
		}
		return 0
	}
	return -1
}

// Delete objects in local but not in COS
func (client *Client) remoteToLocalSyncDelete(localPath string, cosPath string) (ret, successNum, failNum int) {
	q := []PathPair{
		{
			LocalPath: localPath,
			CosPath:   cosPath,
		},
	}
	successNum, failNum = 0, 0
	// BFS folder
	for len(q) > 0 {
		localPath := q[0].LocalPath
		cosPath := q[0].CosPath
		q = q[1:]
		if !strings.HasSuffix(cosPath, "/") {
			cosPath += "/"
		}
		if !strings.HasSuffix(localPath, "/") {
			localPath += "/"
		}
		cosPath = strings.TrimLeft(cosPath, "/")
		files, err := ioutil.ReadDir(localPath)
		if err != nil {
			log.Warn(err.Error())
			return -1, successNum, failNum
		}
		for _, file := range files {
			filePath := path.Join(localPath, file.Name())
			if file.IsDir() {
				q = append(q, PathPair{
					LocalPath: filePath,
					CosPath:   cosPath + file.Name(),
				})
			} else {
				resp, err := client.Client.Object.Head(context.Background(), cosPath+file.Name(), nil)
				if resp != nil && resp.StatusCode == 404 {
					err = os.Remove(filePath)
					if err != nil {
						log.Infof("Delete %s fail", filePath)
						failNum++
					} else {
						log.Infof("Delete %s", filePath)
						successNum++
					}
				} else if err != nil {
					log.Warn(err.Error())
					return -1, successNum, failNum
				}
			}
		}
	}
	return 0, successNum, failNum
}

func (client *Client) remoteToLocalSyncCheck(cosPath string, localPath string, options *DownloadOption) int {
	// check this path is in ignore or include list
	isInclude, isIgnore := false, false
	for _, rule := range options.Include {
		if fnmatch.Match(rule, cosPath, 0) {
			isInclude = true
			break
		}
	}
	for _, rule := range options.Ignore {
		if fnmatch.Match(rule, cosPath, 0) {
			isIgnore = true
			break
		}
	}
	if !isInclude || isIgnore {
		log.Debugf("Skip cos://%s/%s => %s",
			client.Config.Bucket, cosPath, localPath)
		return -2
	}
	if !options.Force {
		if coshelper.IsFile(localPath) {
			if options.Sync {
				resp, err := client.Client.Object.Head(context.Background(), cosPath, nil)
				if err != nil {
					log.Warn(err.Error())
					return -1
				}
				if resp.StatusCode != 200 {
					log.Warnf("Object HEAD Response Code: %d", resp.StatusCode)
					return -1
				}
				md5 := resp.Header.Get("x-cos-meta-md5")
				localMd5 := coshelper.GetFileMd5(localPath)
				size, _ := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
				localSize, _ := coshelper.GetFileSize(localPath)
				if (options.SkipMd5 || md5 == localMd5) && size == localSize {
					log.Debugf("Skip cos://%s/%s => %s",
						client.Config.Bucket, cosPath, localPath)
					return -2
				}
			} else {
				log.Warnf("The file %s already exists, please use -f to overwrite the file",
					localPath)
				return -1
			}
		}
	}
	return 0
}
