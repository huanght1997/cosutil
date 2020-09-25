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

	"github.com/huanght1997/cosutil/cli"
	"github.com/huanght1997/cosutil/coshelper"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	moveCmd = &cobra.Command{
		DisableFlagsInUseLine: true,
		Use: "move [-h] [-H HEADERS] [-d {Copy, Replaced}] [-r]" +
			" [--include INCLUDE] [--ignore IGNORE] SOURCE_PATH COS_PATH",
		Short: "Move file from COS to COS",
		Long: `Move file from COS to COS

SOURCE_PATH	Source file path as 'bucket-appid.cos.ap-guangzhou.myqcloud.com/a.txt'
COS_PATH	COS Path as a/b.txt`,
		Args: cobra.ExactArgs(2),
		RunE: move,
	}
)

func init() {
	rootCmd.AddCommand(moveCmd)

	moveCmd.Flags().SortFlags = false
	moveCmd.Flags().StringVarP(&copyConfig.headers, "headers", "H", "{}", "Specify HTTP headers")
	moveCmd.Flags().StringVarP(&copyConfig.directive, "directive", "d", "Copy", "if Overwrite headers")
	moveCmd.Flags().BoolVarP(&copyConfig.recursive, "recursive", "r", false, "Move files recursively")
	moveCmd.Flags().StringVar(&copyConfig.include, "include", "*",
		"Specify filter rules, separated by commas; Example: *.txt,*.docx,*.ppt")
	moveCmd.Flags().StringVar(&copyConfig.ignore, "ignore", "",
		"Specify ignored rules, separated by commas; Example: *.txt,*.docx,*.ppt")
}

func move(_ *cobra.Command, args []string) error {
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
		Sync:      false,
		Force:     true,
		Directive: copyConfig.directive,
		SkipMd5:   true,
		Ignore:    strings.Split(copyConfig.ignore, ","),
		Include:   strings.Split(copyConfig.include, ","),
		Delete:    false,
		Move:      true,
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
		switch ret {
		case 0:
			return nil
		case -3:
			log.Info("Sync folder canceled by user")
			return nil
		default:
			return coshelper.Error{
				Code:    -1,
				Message: "move folder failed",
			}
		}
	} else {
		ret := client.CopyFile(args[0], cosPath, headers, options)
		switch ret {
		case 0:
			return nil
		case -2:
			log.Info("move folder canceled")
			return nil
		default:
			return coshelper.Error{
				Code:    -1,
				Message: "move file failed",
			}
		}
	}
}
