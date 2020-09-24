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
	"cosutil/coshelper"

	"github.com/spf13/cobra"
)

var (
	createBucketCmd = &cobra.Command{
		DisableFlagsInUseLine: true,
		Use:                   "createbucket [-h] [BUCKET_NAME]",
		Short:                 "Create Bucket",
		Long: `Create Bucket

BUCKET_NAME	name of bucket without appid, the config bucket name will be used if not specified.`,
		Args: cobra.MaximumNArgs(1),
		RunE: createBucket,
	}
)

func init() {
	rootCmd.AddCommand(createBucketCmd)
}

func createBucket(_ *cobra.Command, args []string) error {
	conf := cli.LoadConf(cli.ConfigPath)
	if len(args) > 0 {
		appidIndex := strings.LastIndex(conf.Bucket, "-")
		conf.Bucket = args[0] + conf.Bucket[appidIndex:]
	}
	client := cli.NewClient(conf)
	if client.CreateBucket() {
		return nil
	} else {
		return coshelper.Error{
			Code:    -1,
			Message: "create bucket fail",
		}
	}
}
