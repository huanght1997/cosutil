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

// PutBucketACLConfig defines ACL for an object.
type PutBucketACLConfig struct {
	grantRead, grantWrite, grantFullControl string
}

var (
	putBucketACLConfig PutBucketACLConfig
	putBucketACLCmd    = &cobra.Command{
		DisableFlagsInUseLine: true,
		Use: "putbucketacl [-h] [--grant-read GRANT_READ] [--grant-write GRANT_WRITE]" +
			" [--grant-full-control GRANT_FULL_CONTROL] COS_PATH",
		Short: "Set bucket ACL",
		Args:  cobra.ExactArgs(0),
		RunE:  putBucketACL,
	}
)

func init() {
	rootCmd.AddCommand(putBucketACLCmd)

	putBucketACLCmd.Flags().SortFlags = false
	putBucketACLCmd.Flags().StringVar(&putBucketACLConfig.grantRead, "grant-read", "",
		"Set grant-read")
	putBucketACLCmd.Flags().StringVar(&putBucketACLConfig.grantWrite, "grant-write", "",
		"Set grant-write")
	putBucketACLCmd.Flags().StringVar(&putBucketACLConfig.grantFullControl, "grant-full-control", "",
		"Set grant-full-control")
}

func putBucketACL(_ *cobra.Command, args []string) error {
	conf := cli.LoadConf(cli.ConfigPath)
	client := cli.NewClient(conf)
	cosPath := strings.TrimLeft(args[0], "/")
	if client.PutBucketACL(putBucketACLConfig.grantRead, putBucketACLConfig.grantWrite,
		putBucketACLConfig.grantFullControl, cosPath) {
		return nil
	}
	return coshelper.Error{
		Code:    -1,
		Message: "Put bucket acl fail",
	}
}
