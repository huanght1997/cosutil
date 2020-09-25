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

	"github.com/spf13/cobra"
)

type ListConfig struct {
	all, recursive, versions, human bool
	num                             int
}

var (
	listConfig ListConfig
	listCmd    = &cobra.Command{
		DisableFlagsInUseLine: true,
		Use:                   "list [-h] [-a] [-r] [-n NUM] [-v] [--human] [COS_PATH]",
		Short:                 "List files on COS",
		Long: `List files on COS

[COS_PATH]	COS path as a/b.txt`,
		Args: cobra.MaximumNArgs(1),
		RunE: cosList,
	}
)

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().SortFlags = false
	listCmd.Flags().BoolVarP(&listConfig.all, "all", "a", false, "List all the files")
	listCmd.Flags().BoolVarP(&listConfig.recursive, "recursive", "r", false, "List files recursively")
	listCmd.Flags().IntVarP(&listConfig.num, "num", "n", 100, "Specify max num of files to list")
	listCmd.Flags().BoolVarP(&listConfig.versions, "versions", "v", false, "List objects with versions")
	listCmd.Flags().BoolVar(&listConfig.human, "human", false, "Humanized display")
}

func cosList(_ *cobra.Command, args []string) error {
	cosPath := ""
	if len(args) > 0 {
		cosPath = args[0]
	}
	conf := cli.LoadConf(cli.ConfigPath)
	client := cli.NewClient(conf)
	cosPath = strings.TrimLeft(cosPath, "/")
	options := &cli.ListOption{
		Recursive: listConfig.recursive,
		All:       listConfig.all,
		Num:       listConfig.num,
		Human:     listConfig.human,
		Versions:  listConfig.versions,
	}
	if !client.ListObjects(cosPath, options) {
		return coshelper.Error{
			Code:    -1,
			Message: "list objects failed",
		}
	}
	return nil
}
