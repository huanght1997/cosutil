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

	"cosutil/cli"
	. "cosutil/coshelper"

	"github.com/spf13/cobra"
)

type PutBucketAclConfig struct {
	grantRead, grantWrite, grantFullControl string
}

var (
	putBucketAclConfig PutBucketAclConfig
	putBucketAclCmd    = &cobra.Command{
		DisableFlagsInUseLine: true,
		Use: "putbucketacl [-h] [--grant-read GRANT_READ] [--grant-write GRANT_WRITE]" +
			" [--grant-full-control GRANT_FULL_CONTROL] COS_PATH",
		Short: "Set bucket ACL",
		Args:  cobra.ExactArgs(0),
		RunE:  putBucketAcl,
	}
)

func init() {
	rootCmd.AddCommand(putBucketAclCmd)

	putBucketAclCmd.Flags().SortFlags = false
	putBucketAclCmd.Flags().StringVar(&putBucketAclConfig.grantRead, "grant-read", "",
		"Set grant-read")
	putBucketAclCmd.Flags().StringVar(&putBucketAclConfig.grantWrite, "grant-write", "",
		"Set grant-write")
	putBucketAclCmd.Flags().StringVar(&putBucketAclConfig.grantFullControl, "grant-full-control", "",
		"Set grant-full-control")
}

func putBucketAcl(_ *cobra.Command, args []string) error {
	conf := cli.LoadConf(cli.ConfigPath)
	client := cli.NewClient(conf)
	cosPath := strings.TrimLeft(args[0], "/")
	if client.PutBucketAcl(putBucketAclConfig.grantRead, putBucketAclConfig.grantWrite,
		putBucketAclConfig.grantFullControl, cosPath) {
		return nil
	} else {
		return Error{
			Code:    -1,
			Message: "Put bucket acl fail",
		}
	}
}
