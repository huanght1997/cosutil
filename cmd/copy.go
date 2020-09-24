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
	"strings"

	"cosutil/cli"
	"cosutil/coshelper"

	"github.com/spf13/cobra"
)

type CopyConfig struct {
	sync, recursive, force, skipMd5, deleteTarget bool
	headers, include, ignore, directive           string
}

var (
	copyConfig CopyConfig
	copyCmd    = &cobra.Command{
		DisableFlagsInUseLine: true,
		Use:                   "copy [-h] [-H HEADERS] [-d {Copy,Replaced}] [-s] [-r] [-f] [--include INCLUDE] [--ignore IGNORE] [--skipmd5] [--delete] SOURCE_PATH COS_PATH",
		Short:                 "Copy file from COS to COS",
		Long: `Copy file from COS to COS

SOURCE_PATH	Source file path as 'bucket-appid.cos.ap-guangzhou.myqcloud.com/a.txt'
COS_PATH	COS Path as a/b.txt`,
		Args: cobra.ExactArgs(2),
		RunE: copyCos,
	}
)

func init() {
	rootCmd.AddCommand(copyCmd)

	copyCmd.Flags().SortFlags = false
	copyCmd.Flags().StringVarP(&copyConfig.headers, "headers", "H", "{}", "Specify HTTP headers")
	copyCmd.Flags().StringVarP(&copyConfig.directive, "directive", "d", "Copy", "if Overwrite headers")
	copyCmd.Flags().BoolVarP(&copyConfig.sync, "sync", "s", false, "Copy and skip the same file")
	copyCmd.Flags().BoolVarP(&copyConfig.recursive, "recursive", "r", false, "Copy files recursively")
	copyCmd.Flags().BoolVarP(&copyConfig.force, "force", "f", false, "Overwrite file without skip")
	copyCmd.Flags().StringVar(&copyConfig.include, "include", "*",
		"Specify filter rules, separated by commas; Example: *.txt,*.docx,*.ppt")
	copyCmd.Flags().StringVar(&copyConfig.ignore, "ignore", "",
		"Specify ignored rules, separated by commas; Example: *.txt,*.docx,*.ppt")
	copyCmd.Flags().BoolVar(&copyConfig.skipMd5, "skipmd5", false,
		"Copy sync without md5 check, only check filename and filesize")
	copyCmd.Flags().BoolVar(&copyConfig.deleteTarget, "delete", false,
		"Delete objects whick exists in source path but not exist in dest path")
}

func copyCos(_ *cobra.Command, args []string) error {
	conf := cli.LoadConf(cli.ConfigPath)
	client := cli.NewClient(conf)
	_, cosPath := concatPath(args[0], args[1])
	for strings.HasPrefix(cosPath, "/") {
		cosPath = cosPath[1:]
	}
	if copyConfig.directive != "Copy" && copyConfig.directive != "Replaced" {
		return coshelper.Error{
			Code:    1,
			Message: "-d/--directive flags must be 'Copy' or 'Replaced'",
		}
	}
	options := &cli.CopyOption{
		Sync:      copyConfig.sync,
		Force:     copyConfig.force,
		Directive: copyConfig.directive,
		SkipMd5:   copyConfig.skipMd5,
		Ignore:    strings.Split(copyConfig.ignore, ","),
		Include:   strings.Split(copyConfig.include, ","),
		Delete:    copyConfig.deleteTarget,
		Move:      false,
	}
	headers := coshelper.ConvertStringToHeader(copyConfig.headers)
	if copyConfig.recursive {
		_, cosPath = concatPath(args[0], cosPath)
		if !strings.HasSuffix(cosPath, "/") {
			cosPath += "/"
		}
		if strings.HasPrefix(cosPath, "/") {
			cosPath = cosPath[1:]
		}
		ret := client.CopyFolder(args[0], cosPath, headers, options)
		if ret == 0 {
			return nil
		} else {
			return coshelper.Error{
				Code:    ret,
				Message: "copy folder failed",
			}
		}
	} else {
		ret := client.CopyFile(args[0], cosPath, headers, options)
		if ret == 0 || ret == -2 {
			return nil
		} else {
			return coshelper.Error{
				Code:    ret,
				Message: "copy file failed",
			}
		}
	}
}
