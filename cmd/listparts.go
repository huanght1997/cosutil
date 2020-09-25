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
	"github.com/huanght1997/cosutil/cli"
	"github.com/huanght1997/cosutil/coshelper"

	"github.com/spf13/cobra"
)

var (
	listpartsCmd = &cobra.Command{
		DisableFlagsInUseLine: true,
		Use:                   "listparts [-h] [COS_PATH]",
		Short:                 "List upload parts",
		Long: `List upload parts

COS_PATH	COS Path as a/b.txt`,
		Args: cobra.MaximumNArgs(1),
		RunE: listPart,
	}
)

func init() {
	rootCmd.AddCommand(listpartsCmd)
}

func listPart(_ *cobra.Command, args []string) error {
	cosPath := ""
	if len(args) >= 1 {
		cosPath = args[0]
	}
	conf := cli.LoadConf(cli.ConfigPath)
	client := cli.NewClient(conf)
	if !client.ListMultipartObjects(cosPath) {
		return coshelper.Error{
			Code:    -1,
			Message: "list multipart object failed",
		}
	}
	return nil
}
