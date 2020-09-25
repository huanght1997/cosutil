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
	putBucketVersioningCmd = &cobra.Command{
		DisableFlagsInUseLine: true,
		Use:                   "putbucketversioning [-h] {Enabled,Suspended}",
		Short:                 "Set the versioning state",
		Long: `Set the versioning state

{Enabled,Suspended}	Status of bucket`,
		Args: cobra.ExactArgs(1),
		RunE: putBucketVersioning,
	}
)

func init() {
	rootCmd.AddCommand(putBucketVersioningCmd)
}

func putBucketVersioning(_ *cobra.Command, args []string) error {
	var versioning bool
	switch strings.ToLower(args[0]) {
	case "enabled":
		versioning = true
	case "suspended":
		versioning = false
	default:
		return coshelper.Error{
			Code:    1,
			Message: "invalid argument, must be one of them: Enabled or Suspended",
		}
	}
	conf := cli.LoadConf(cli.ConfigPath)
	client := cli.NewClient(conf)
	if !client.PutBucketVersioning(versioning) {
		return coshelper.Error{
			Code:    -1,
			Message: "put bucket versioning fail",
		}
	}
	return nil
}
