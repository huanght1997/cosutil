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

// PutObjectACLConfig defines ACL config for object.
type PutObjectACLConfig struct {
	grantRead, grantWrite, grantFullControl string
}

var (
	putObjectACLConfig PutObjectACLConfig
	putObjectACLCmd    = &cobra.Command{
		DisableFlagsInUseLine: true,
		Use: "putobjectacl [-h] [--grant-read GRANT_READ] [--grant-write GRANT_WRITE]" +
			" [--grant-full-control GRANT_FULL_CONTROL] COS_PATH",
		Short: "Set object ACL",
		Long: `Set object ACL

COS_PATH	COS Path as a/b.txt`,
		Args: cobra.ExactArgs(1),
		RunE: putObjectACL,
	}
)

func init() {
	rootCmd.AddCommand(putObjectACLCmd)

	putObjectACLCmd.Flags().SortFlags = false
	putObjectACLCmd.Flags().StringVar(&putObjectACLConfig.grantRead, "grant-read", "",
		"Set grant-read")
	putObjectACLCmd.Flags().StringVar(&putObjectACLConfig.grantWrite, "grant-write", "",
		"Set grant-write")
	putObjectACLCmd.Flags().StringVar(&putObjectACLConfig.grantFullControl, "grant-full-control", "",
		"Set grant-full-control")
}

func putObjectACL(_ *cobra.Command, args []string) error {
	conf := cli.LoadConf(cli.ConfigPath)
	client := cli.NewClient(conf)
	cosPath := strings.TrimLeft(args[0], "/")
	if client.PutObjectACL(putObjectACLConfig.grantRead, putObjectACLConfig.grantWrite,
		putObjectACLConfig.grantFullControl, cosPath) {
		return nil
	}
	return coshelper.Error{
		Code:    -1,
		Message: "Put object acl fail",
	}
}
