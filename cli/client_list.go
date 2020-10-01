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
	"os"

	"github.com/huanght1997/cosutil/coshelper"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	log "github.com/sirupsen/logrus"
	"github.com/tencentyun/cos-go-sdk-v5"
)

type ListOption struct {
	Recursive bool
	All       bool
	Num       int
	Human     bool
	Versions  bool
}

type FileDesc struct {
	Path      string
	Type      string
	Size      int
	Time      string
	Class     string
	VersionID string
}

func (client *Client) ListObjects(cosPath string, options *ListOption) bool {
	isTruncated := true
	delimiter := "/"
	if options.Recursive {
		delimiter = ""
	}
	if options.All {
		options.Num = -1
	}
	fileNum := 0
	totalSize := 0
	keyMarker := ""
	versionIDMarker := ""
	for isTruncated {
		filesInfo := make([]FileDesc, 0)
		for i := 0; i <= client.Config.RetryTimes; i++ {
			if versionIDMarker == "null" {
				versionIDMarker = ""
			}
			var resp *cos.Response
			var err error
			var res interface{}
			if options.Versions {
				res, resp, err = client.Client.Bucket.GetObjectVersions(context.Background(), &cos.BucketGetObjectVersionsOptions{
					Prefix:          cosPath,
					Delimiter:       delimiter,
					KeyMarker:       keyMarker,
					VersionIdMarker: versionIDMarker,
					MaxKeys:         1000,
				})
			} else {
				res, resp, err = client.Client.Bucket.Get(context.Background(), &cos.BucketGetOptions{
					Prefix:    cosPath,
					Delimiter: delimiter,
					Marker:    keyMarker,
					MaxKeys:   1000,
				})
			}
			if resp != nil && resp.StatusCode != 200 {
				respContent, _ := ioutil.ReadAll(resp.Body)
				log.Warn("List Object Version Response Code: %d, Response Content: %s",
					resp.StatusCode, string(respContent))
			} else if err != nil {
				log.Warn(err.Error())
			} else {
				if options.Versions {
					result := res.(*cos.BucketGetObjectVersionsResult)
					isTruncated = result.IsTruncated
					keyMarker = result.NextKeyMarker
					versionIDMarker = result.NextVersionIdMarker
					for _, folder := range result.CommonPrefixes {
						filesInfo = append(filesInfo, FileDesc{
							Path:      folder,
							Type:      "DIR",
							Time:      "",
							VersionID: "",
						})
					}
					for _, file := range result.DeleteMarker {
						fileNum++
						filesInfo = append(filesInfo, FileDesc{
							Path:      file.Key,
							Type:      "",
							Time:      coshelper.ConvertTime(file.LastModified),
							VersionID: file.VersionId,
						})
						if fileNum == options.Num {
							break
						}
					}
					if fileNum < options.Num || options.Num == -1 {
						for _, file := range result.Version {
							fileNum++
							totalSize += file.Size
							filesInfo = append(filesInfo, FileDesc{
								Path:      file.Key,
								Type:      "File",
								Size:      file.Size,
								Time:      coshelper.ConvertTime(file.LastModified),
								VersionID: file.VersionId,
							})
							if fileNum == options.Num {
								break
							}
						}
					}
					break
				} else {
					result := res.(*cos.BucketGetResult)
					isTruncated = result.IsTruncated
					keyMarker = result.NextMarker
					for _, folder := range result.CommonPrefixes {
						filesInfo = append(filesInfo, FileDesc{
							Path:  folder,
							Type:  "DIR",
							Time:  "",
							Class: "",
						})
					}
					for _, file := range result.Contents {
						fileNum++
						totalSize += file.Size
						filesInfo = append(filesInfo, FileDesc{
							Path:  file.Key,
							Type:  "File",
							Size:  file.Size,
							Time:  coshelper.ConvertTime(file.LastModified),
							Class: file.StorageClass,
						})
						if fileNum == options.Num {
							break
						}
					}
				}
				break
			}
			if i == client.Config.RetryTimes {
				return false
			}
		}
		printFilesInfo(filesInfo, options)
		if fileNum >= options.Num {
			break
		}
	}

	if options.Recursive {
		log.Infof(" Files num: %d", fileNum)
		log.Infof(" Files size: %s", coshelper.Humanize(totalSize, options.Human))
	}
	if !options.All && fileNum == options.Num {
		log.Infof("Has listed the first %d, use '-a' option to list all please",
			fileNum)
	}
	return true
}

func printFilesInfo(filesInfo []FileDesc, options *ListOption) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateColumns = false
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, Align: text.AlignLeft},
		{Number: 2, Align: text.AlignRight},
		{Number: 3, Align: text.AlignLeft},
		{Number: 4, Align: text.AlignLeft},
	})
	if options.Versions {
		for _, row := range filesInfo {
			if row.Type == "File" {
				t.AppendRow(table.Row{
					row.Path,
					coshelper.Humanize(row.Size, options.Human),
					row.Time,
					row.VersionID,
				})
			} else {
				t.AppendRow(table.Row{
					row.Path,
					row.Type,
					row.Time,
					row.VersionID,
				})
			}
		}
	} else {
		for _, row := range filesInfo {
			if row.Type == "File" {
				t.AppendRow(table.Row{
					row.Path,
					coshelper.Humanize(row.Size, options.Human),
					row.Class,
					row.Time,
				})
			} else {
				t.AppendRow(table.Row{
					row.Path,
					row.Type,
					row.Class,
					row.Time,
				})
			}
		}
	}
	t.Render()
}
