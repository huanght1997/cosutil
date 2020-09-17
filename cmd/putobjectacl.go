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
	"cosutil/cli"
	. "cosutil/coshelper"

	"github.com/spf13/cobra"
	"strings"
)

type PutObjectAclConfig struct {
	grantRead, grantWrite, grantFullControl string
}

var (
	putObjectAclConfig PutObjectAclConfig
	putObjectAclCmd    = &cobra.Command{
		DisableFlagsInUseLine: true,
		Use: "putobjectacl [-h] [--grant-read GRANT_READ] [--grant-write GRANT_WRITE]" +
			" [--grant-full-control GRANT_FULL_CONTROL] COS_PATH",
		Short: "Set object ACL",
		Long: `Set object ACL

COS_PATH	COS Path as a/b.txt`,
		Args: cobra.ExactArgs(1),
		RunE: putObjectAcl,
	}
)

func init() {
	rootCmd.AddCommand(putObjectAclCmd)

	putObjectAclCmd.Flags().SortFlags = false
	putObjectAclCmd.Flags().StringVar(&putObjectAclConfig.grantRead, "grant-read", "",
		"Set grant-read")
	putObjectAclCmd.Flags().StringVar(&putObjectAclConfig.grantWrite, "grant-write", "",
		"Set grant-write")
	putObjectAclCmd.Flags().StringVar(&putObjectAclConfig.grantFullControl, "grant-full-control", "",
		"Set grant-full-control")
}

func putObjectAcl(_ *cobra.Command, args []string) error {
	conf := cli.LoadConf(cli.ConfigPath)
	client := cli.NewClient(conf)
	cosPath := strings.TrimLeft(args[0], "/")
	if client.PutObjectAcl(putObjectAclConfig.grantRead, putObjectAclConfig.grantWrite,
		putObjectAclConfig.grantFullControl, cosPath) {
		return nil
	} else {
		return Error{
			Code:    -1,
			Message: "Put object acl fail",
		}
	}
}
