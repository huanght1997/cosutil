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
)

var (
	forceDeleteBucket bool
	deleteBucketCmd   = &cobra.Command{
		DisableFlagsInUseLine: true,
		Use:                   "deletebucket [-h] [-f]",
		Short:                 "Delete Bucket",
		RunE:                  deleteBucket,
	}
)

func init() {
	rootCmd.AddCommand(deleteBucketCmd)

	deleteBucketCmd.Flags().BoolVarP(&forceDeleteBucket, "force", "f", false,
		"Clear all inside the bucket and delete bucket")
}

func deleteBucket(*cobra.Command, []string) error {
	conf := cli.LoadConf(cli.ConfigPath)
	client := cli.NewClient(conf)
	if client.DeleteBucket(forceDeleteBucket) {
		return nil
	}
	return Error{
		Code:    -1,
		Message: "delete bucket fail",
	}
}
