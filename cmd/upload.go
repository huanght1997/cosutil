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
	"fmt"
	"os"
	"strings"

	"github.com/huanght1997/cosutil/cli"
	"github.com/huanght1997/cosutil/coshelper"

	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type UploadConfig struct {
	recursive, sync, force, yes, skipMd5, delRemote bool
	headers, include, ignore                        string
}

// uploadCmd represents the upload command
var (
	uploadLocalPath, uploadCosPath string
	uploadCmd                      = &cobra.Command{
		DisableFlagsInUseLine: true,
		Use:                   "upload [-h] [-r] [-H HEADERS] [-s] [-f] [--include INCLUDE] [--ignore IGNORE] [--skipmd5] [--delete] LOCAL_PATH COS_PATH",
		Short:                 "Upload file or directory to COS",
		Long: `Upload file or directory to COS.

LOCAL_PATH	Local file path as /tmp/a.txt or directory
COS_PATH	COS path as a/b.txt`,
		Args: cobra.ExactArgs(2),
		RunE: upload,
	}
	uploadConfig UploadConfig
)

func init() {
	rootCmd.AddCommand(uploadCmd)

	uploadCmd.Flags().SortFlags = false
	uploadCmd.Flags().BoolVarP(&uploadConfig.recursive, "recursive", "r", false,
		"Upload recursively when upload directory")
	uploadCmd.Flags().StringVarP(&uploadConfig.headers, "headers", "H", "{}",
		"Specify HTTP headers")
	uploadCmd.Flags().BoolVarP(&uploadConfig.sync, "sync", "s", false,
		"Upload and skip the same file")
	uploadCmd.Flags().BoolVarP(&uploadConfig.force, "force", "f", false,
		"Upload without history breakpoint")
	uploadCmd.Flags().BoolVarP(&uploadConfig.yes, "yes", "y", false,
		"Skip confirmation")
	uploadCmd.Flags().StringVar(&uploadConfig.include, "include", "*",
		"Specify filter rules, separated by commas; Example: *.txt,*.docx,*.ppt")
	uploadCmd.Flags().StringVar(&uploadConfig.ignore, "ignore", "",
		"Specify ignored rules, separated by commas; Example: *.txt,*.docx,*.ppt")
	uploadCmd.Flags().BoolVar(&uploadConfig.skipMd5, "skipmd5", false,
		"Upload without x-cos-meta-md5 / sync without check md5, only check filename and filesize")
	uploadCmd.Flags().BoolVar(&uploadConfig.delRemote, "delete", false,
		"Delete objects which exists in COS but not exist in local")
}

func upload(_ *cobra.Command, args []string) error {
	uploadLocalPath, _ = homedir.Expand(args[0])
	uploadCosPath = args[1]
	conf := cli.LoadConf(cli.ConfigPath)
	client := cli.NewClient(conf)
	// remove prefix slashes
	uploadCosPath = strings.TrimLeft(uploadCosPath, "/")
	if uploadCosPath == "" {
		uploadCosPath = "/"
	}

	if !coshelper.FileExists(uploadLocalPath) {
		log.Warnf("cannot stat '%s': No such file or directory", uploadLocalPath)
		return coshelper.Error{
			Code:    1,
			Message: "no such file or directory",
		}
	}

	f, err := os.OpenFile(uploadLocalPath, os.O_RDONLY, 0755)
	if err != nil {
		log.Warnf("local path '%s' is not readable!", uploadLocalPath)
		return coshelper.Error{
			Code:    1,
			Message: "local file not readable",
		}
	}
	_ = f.Close()

	uploadLocalPath, uploadCosPath = concatPath(uploadLocalPath, uploadCosPath)
	uploadCosPath = strings.TrimPrefix(uploadCosPath, "/")
	uploadOption := &cli.UploadOption{
		SkipMd5: uploadConfig.skipMd5,
		Sync:    uploadConfig.sync,
		Include: strings.Split(uploadConfig.include, ","),
		Ignore:  strings.Split(uploadConfig.ignore, ","),
		Force:   uploadConfig.force,
		Yes:     uploadConfig.yes,
		Delete:  uploadConfig.delRemote,
	}
	headers := coshelper.ConvertStringToHeader(uploadConfig.headers)
	if uploadConfig.recursive {
		var ret int
		if coshelper.IsFile(uploadLocalPath) {
			ret = client.UploadFile(uploadLocalPath, uploadCosPath, headers, uploadOption)
		} else if coshelper.IsDir(uploadLocalPath) {
			ret = client.UploadFolder(uploadLocalPath, uploadCosPath, headers, uploadOption)
		}
		switch ret {
		case 0:
			return nil
		case -2:
			log.Info("This file has existed in COS. Skipped.")
			return nil
		default:
			return coshelper.Error{
				Code:    ret,
				Message: fmt.Sprintf("upload failed, code: %d", ret),
			}
		}
	} else {
		if coshelper.IsDir(uploadLocalPath) {
			log.Warnf(`"%s" is a directory, use '-r' option to upload it please`, uploadLocalPath)
			return coshelper.Error{
				Code:    1,
				Message: "upload directory without -r option",
			}
		} else if !coshelper.IsFile(uploadLocalPath) {
			log.Warnf("cannot stat '%s': No such file or directory", uploadLocalPath)
			return coshelper.Error{
				Code:    1,
				Message: "cannot access file",
			}
		}
		ret := client.UploadFile(uploadLocalPath, uploadCosPath, headers, uploadOption)
		switch ret {
		case 0:
			return nil
		case -2:
			log.Info("This file has existed in COS. Skipped.")
			return nil
		default:
			return coshelper.Error{
				Code:    ret,
				Message: fmt.Sprintf("upload failed, code: %d", ret),
			}
		}
	}
}

// if sourcePath is a file, targetPath is a directory, append file name to targetPath.
func concatPath(sourcePath string, targetPath string) (source, target string) {
	source = strings.ReplaceAll(sourcePath, "\\", "/")
	target = strings.ReplaceAll(targetPath, "\\", "/")
	if !strings.HasSuffix(source, "/") && strings.HasSuffix(target, "/") {
		offset := strings.LastIndex(source, "/") + 1
		target += source[offset:]
	}
	return
}
