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
package cmd

import (
	log "github.com/sirupsen/logrus"
	"strings"

	"cosutil/cli"
	. "cosutil/coshelper"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

type DownloadConfig struct {
	force, recursive, sync, skipMd5, delLocal bool
	headers, versionId, include, ignore       string
	num                                       int
}

var (
	downloadCosPath, downloadLocalPath string
	downloadConfig                     DownloadConfig
	downloadCmd                        = &cobra.Command{
		DisableFlagsInUseLine: true,
		Use: "download [-h] [-f] [-r] [-s] [-H HEADERS] [--versionId VERSIONID] [--include INCLUDE] " +
			"[--ignore IGNORE] [--skipmd5] [--delete] [-n NUM] COS_PATH LOCAL_PATH",
		Short: "Download file or directory from COS.",
		Long: `Download file or directory from COS.

COS_PATH	COS Path as a/b.txt
LOCAL_PATH	Local file path as /tmp/a.txt`,
		Args: cobra.ExactArgs(2),
		RunE: download,
	}
)

func init() {
	rootCmd.AddCommand(downloadCmd)

	downloadCmd.Flags().SortFlags = false
	downloadCmd.Flags().BoolVarP(&downloadConfig.force, "force", "f", false,
		"Overwrite the saved files")
	downloadCmd.Flags().BoolVarP(&downloadConfig.recursive, "recursive", "r", false,
		"Download recursively when upload directory")
	downloadCmd.Flags().BoolVarP(&downloadConfig.sync, "sync", "s", false,
		"Download and skip the same file")
	downloadCmd.Flags().StringVarP(&downloadConfig.headers, "headers", "H", "{}",
		"Specify HTTP headers")
	downloadCmd.Flags().StringVar(&downloadConfig.versionId, "versionId", "",
		"Specify versionId of object to list")
	downloadCmd.Flags().StringVar(&downloadConfig.include, "include", "*",
		"Specify filter rules, separated by commas: Example: *.txt,*.docx,*.ppt")
	downloadCmd.Flags().StringVar(&downloadConfig.ignore, "ignore", "",
		"Specify ignored rules, separated by commas; Example: *.txt,*.docx,*.ppt")
	downloadCmd.Flags().BoolVar(&downloadConfig.skipMd5, "skipmd5", false,
		"Download sync without check md5, only check filename and filesize")
	downloadCmd.Flags().BoolVar(&downloadConfig.delLocal, "delete", false,
		"Delete objects which exists in local but not exist in cos")
	downloadCmd.Flags().IntVarP(&downloadConfig.num, "num", "n", 10,
		"Specify max part num of multidownload")
}

func download(_ *cobra.Command, args []string) error {
	downloadLocalPath, _ = homedir.Expand(args[1])
	downloadCosPath = args[0]
	conf := cli.LoadConf(cli.ConfigPath)
	client := cli.NewClient(conf)
	downloadCosPath, downloadLocalPath = concatPath(downloadCosPath, downloadLocalPath)
	if strings.HasPrefix(downloadCosPath, "/") {
		downloadCosPath = downloadCosPath[1:]
	}
	options := &cli.DownloadOption{
		Force:   downloadConfig.force,
		Sync:    downloadConfig.sync,
		Num:     downloadConfig.num,
		Ignore:  strings.Split(downloadConfig.ignore, ","),
		Include: strings.Split(downloadConfig.include, ","),
		SkipMd5: downloadConfig.skipMd5,
		Delete:  downloadConfig.delLocal,
	}
	if options.Num > 20 {
		options.Num = 20
	}
	headers := ConvertStringToHeader(downloadConfig.headers)
	var rt int
	if downloadConfig.recursive {
		rt = client.DownloadFolder(downloadCosPath, downloadLocalPath, options)
	} else {
		rt = client.DownloadFile(downloadCosPath, downloadLocalPath, headers, options)
	}
	switch rt {
	case 0:
		return nil
	case -2:
		log.Info("some files skipped")
		return nil
	case -3:
		log.Info("some operations canceled by user")
		return nil
	default:
		return Error{
			Code:    rt,
			Message: "download failed",
		}
	}
}
