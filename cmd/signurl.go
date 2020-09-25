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
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/huanght1997/cosutil/cli"
	"github.com/huanght1997/cosutil/coshelper"

	"github.com/spf13/cobra"
)

var (
	signUrlTimeout int
	signUrlCmd     = &cobra.Command{
		DisableFlagsInUseLine: true,
		Use:                   "signurl [-h] [-t TIMEOUT] COS_PATH",
		Short:                 "Get download url",
		Long: `Get download url

COS_PATH	COS Path as a/b.txt`,
		Args: cobra.ExactArgs(1),
		RunE: signUrl,
	}
)

func init() {
	rootCmd.AddCommand(signUrlCmd)

	signUrlCmd.Flags().SortFlags = false
	signUrlCmd.Flags().IntVarP(&signUrlTimeout, "timeout", "t", 1000,
		"Specify the signature valid time")
}

func signUrl(cmd *cobra.Command, args []string) error {
	conf := cli.LoadConf(cli.ConfigPath)
	client := cli.NewClient(conf)
	cosPath := strings.TrimLeft(args[0], "/")
	url, err := client.Client.Object.GetPresignedURL(context.Background(),
		http.MethodGet, cosPath, client.Config.SecretID, client.Config.SecretKey,
		time.Duration(signUrlTimeout)*time.Second, nil)
	if err != nil {
		return coshelper.Error{
			Code:    -1,
			Message: err.Error(),
		}
	} else {
		fmt.Println(url)
		return nil
	}
}
