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
	abortCmd = &cobra.Command{
		DisableFlagsInUseLine: true,
		Use:                   "abort [-h] [COS_PATH]",
		Short:                 "Aborts upload parts on COS",
		Long: `Aborts upload parts on COS
COS_PATH	COS path as a/b.txt`,
		Args: cobra.MaximumNArgs(1),
		RunE: abort,
	}
)

func init() {
	rootCmd.AddCommand(abortCmd)
}

func abort(_ *cobra.Command, args []string) error {
	abortCosPath := ""
	if len(args) > 0 {
		abortCosPath = args[0]
	}
	conf := cli.LoadConf(cli.ConfigPath)
	client := cli.NewClient(conf)
	abortCosPath = strings.TrimLeft(abortCosPath, "/")
	if client.AbortParts(abortCosPath) {
		return nil
	}
	return coshelper.Error{
		Code:    -1,
		Message: "Failed to abort parts",
	}
}
