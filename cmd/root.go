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
	"os"

	"cosutil/cli"
	"cosutil/coshelper"

	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cosutil",
	Short: "A command-line tool for Tencent Cloud COS",
	Long: `An easy-to-use but powerful command-line tool for Tencent Cloud COS.
try 'cosutil -h' to get more information.
try 'cosutil sub-command -h' to learn all command usages, like 'cosutil upload -h'`,
	TraverseChildren: true,
	SilenceUsage:     true,
	Version:          cli.VERSION,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Code: -3 user cancelled, -1 runtime failed(network or file permission), 1 user invalid input, 2 WTF
		if e, ok := err.(coshelper.Error); ok {
			os.Exit(e.Code)
		}
		os.Exit(-1)
	}
}

func init() {
	defaultConfigPath := "~/.cos.conf"
	defaultLogPath := "~/.cos.log"
	expandedDefaultConfPath, err := homedir.Expand(defaultConfigPath)
	if err != nil {
		log.Fatal(err)
	}
	expandedDefaultLogPath, err := homedir.Expand(defaultLogPath)
	if err != nil {
		log.Fatal(err)
	}
	cobra.OnInitialize(func() {
		coshelper.InitLogger(cli.LogPath, cli.LogSize, cli.LogBackupCount, cli.DebugMode)
	})
	rootCmd.Flags().SortFlags = false
	rootCmd.Flags().BoolVarP(&cli.DebugMode, "debug", "d", false,
		"Debug mode")
	rootCmd.Flags().StringVarP(&cli.Bucket, "bucket", "b", "",
		"Specify bucket")
	rootCmd.Flags().StringVarP(&cli.Region, "region", "r", "",
		"Specify region")
	rootCmd.Flags().StringVarP(&cli.ConfigPath, "config_path", "c", expandedDefaultConfPath,
		"Specify config path")
	rootCmd.Flags().StringVarP(&cli.LogPath, "log_path", "l", expandedDefaultLogPath,
		"Specify log path")
	rootCmd.Flags().IntVar(&cli.LogSize, "log_size", 1,
		"Specify max log size in MB")
	rootCmd.Flags().IntVar(&cli.LogBackupCount, "log_backup_count", 1,
		"Specify log backup num")
}
