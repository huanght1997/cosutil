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

type RestoreConfig struct {
	recursive bool
	day       int
	tier      string
}

var (
	restoreConfig RestoreConfig
	restoreCmd    = &cobra.Command{
		DisableFlagsInUseLine: true,
		Use:                   "restore [-h] [-r] [-d DAY] [-t {Expedited,Standard,Bulk}] COS_PATH",
		Short:                 "Restore",
		Long: `Restore

COS_PATH	COS Path as a/b.txt`,
		Args: cobra.ExactArgs(1),
		RunE: restore,
	}
)

func init() {
	rootCmd.AddCommand(restoreCmd)

	restoreCmd.Flags().SortFlags = false
	restoreCmd.Flags().BoolVarP(&restoreConfig.recursive, "recursive", "r", false,
		"Restore files recursively")
	restoreCmd.Flags().IntVarP(&restoreConfig.day, "day", "d", 7,
		"Specify lifetime of the restored (active) copy")
	restoreCmd.Flags().StringVarP(&restoreConfig.tier, "tier", "t", "STANDARD",
		"Specify the data access tier")
}

func restore(_ *cobra.Command, args []string) error {
	cosPath := strings.TrimLeft(args[0], "/")
	conf := cli.LoadConf(cli.ConfigPath)
	client := cli.NewClient(conf)
	options := &cli.RestoreOption{
		Day: restoreConfig.day,
	}
	switch strings.ToLower(restoreConfig.tier) {
	case "expedited":
		options.Tier = cli.Expedited
	case "standard":
		options.Tier = cli.Standard
	case "bulk":
		options.Tier = cli.Bulk
	default:
		return coshelper.Error{
			Code:    1,
			Message: "invalid -t option: must be one of them - Expedited, Standard, Bulk",
		}
	}
	if restoreConfig.recursive {
		ret := client.RestoreFolder(cosPath, options)
		if ret == 0 {
			return nil
		} else {
			return coshelper.Error{
				Code:    ret,
				Message: "restore failed",
			}
		}
	} else {
		ret := client.RestoreFile(cosPath, options)
		if ret != 0 {
			return coshelper.Error{
				Code:    ret,
				Message: "restore failed",
			}
		}
		return nil
	}
}
