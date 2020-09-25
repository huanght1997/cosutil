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

var (
	infoCmd = &cobra.Command{
		DisableFlagsInUseLine: true,
		Use:                   "info [-h] [--human] COS_PATH",
		Short:                 "Get the information of file on COS",
		Long: `Get the information of file on COS

COS_PATH	COS Path as a/b.txt`,
		Args: cobra.ExactArgs(1),
		RunE: info,
	}
	human bool
)

func init() {
	rootCmd.AddCommand(infoCmd)

	infoCmd.Flags().SortFlags = false
	infoCmd.Flags().BoolVar(&human, "human", false, "Humanized display")
}

func info(_ *cobra.Command, args []string) error {
	cosPath := strings.TrimLeft(args[0], "/")
	conf := cli.LoadConf(cli.ConfigPath)
	client := cli.NewClient(conf)
	if !client.InfoObject(cosPath, human) {
		return coshelper.Error{
			Code:    -1,
			Message: "info object failed",
		}
	}
	return nil
}
