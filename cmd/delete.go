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
	"errors"
	"fmt"
	"strings"

	"cosutil/cli"
	"cosutil/coshelper"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type DeleteConfig struct {
	recursive, versions, force bool
	versionId                  string
}

var (
	deleteConfig DeleteConfig
	deleteCmd    = &cobra.Command{
		DisableFlagsInUseLine: true,
		Use:                   "delete [-h] [-r] [--versions] [--versionId VERSIONID] [-f] COS_PATH",
		Short:                 "Delete file or files on COS",
		Long: `Delete file or files on COS

COS_PATH	COS path as a/b.txt`,
		Args: cobra.ExactArgs(1),
		RunE: deleteCos,
	}
)

func init() {
	rootCmd.AddCommand(deleteCmd)

	deleteCmd.Flags().SortFlags = false
	deleteCmd.Flags().BoolVarP(&deleteConfig.recursive, "recursive", "r", false,
		"Delete files recursively, WARN: all files with the prefix will be deleted!")
	deleteCmd.Flags().BoolVar(&deleteConfig.versions, "versions", false,
		"Delete objects with versions")
	deleteCmd.Flags().StringVar(&deleteConfig.versionId, "versionId", "",
		"Specify versionId of object to list")
	deleteCmd.Flags().BoolVarP(&deleteConfig.force, "force", "f", false,
		"Delete directly without confirmation")
}

func deleteCos(_ *cobra.Command, args []string) error {
	deleteCosPath := args[0]
	conf := cli.LoadConf(cli.ConfigPath)
	client := cli.NewClient(conf)
	for strings.HasPrefix(deleteCosPath, "/") {
		deleteCosPath = deleteCosPath[1:]
	}
	options := &cli.DeleteOption{
		Force:     deleteConfig.force,
		Versions:  deleteConfig.versions,
		VersionId: deleteConfig.versionId,
	}
	var ret int
	if deleteConfig.recursive {
		if !strings.HasSuffix(deleteCosPath, "/") {
			deleteCosPath += "/"
		}
		if deleteCosPath == "/" {
			deleteCosPath = ""
		}
		ret = client.DeleteFolder(deleteCosPath, options)
	} else {
		if deleteCosPath == "" {
			log.Warn("not support delete empty path")
			return errors.New("not support delete empty path")
		}
		ret = client.DeleteFile(deleteCosPath, options)
	}
	switch ret {
	case 0:
		log.Debugf("delete all files under %s successfully!", deleteCosPath)
		return nil
	case -3:
		log.Infof("delete files under %s canceled by user.", deleteCosPath)
		return nil
	default:
		log.Debugf("delete all files under %s failed!", deleteCosPath)
		return coshelper.Error{
			Code:    ret,
			Message: fmt.Sprintf("delete all files under %s failed!", deleteCosPath),
		}
	}
}
