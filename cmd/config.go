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
	"fmt"
	"github.com/spf13/pflag"
	"strconv"

	"cosutil/cli"
	. "cosutil/coshelper"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
)

var (
	configCmd = &cobra.Command{
		DisableFlagsInUseLine: true,
		Use:                   "config [-h] -a SECRET_ID -s SECRET_KEY [-t TOKEN] -b BUCKET (-r REGION | -e ENDPOINT) [-m MAX_THREAD] [-p PART_SIZE] [--retry RETRY] [--timeout TIMEOUT] [-u APPID] [--verify VERIFY] [--do-not-use-ssl] [--anonymous]",
		Short:                 "Config your information at first",
		RunE:                  config,
	}
	configSecretId, configSecretKey, configToken, configBucket       string
	configRegion, configEndpoint                                     string
	configMaxThread, configPartSize, configRetryTimes, configTimeout int
	configAppId, configVerifyMethod                                  string
	configNoSsl, configAnonymous                                     bool
)

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.Flags().SortFlags = false
	configCmd.Flags().StringVarP(&configSecretId, "secret_id", "a", "", "Specify your secret id")
	_ = configCmd.MarkFlagRequired("secret_id")
	configCmd.Flags().StringVarP(&configSecretKey, "secret_key", "s", "", "Specify your secret key")
	_ = configCmd.MarkFlagRequired("secret_key")
	configCmd.Flags().StringVarP(&configToken, "token", "t", "", "Set x-cos-security-token header")
	configCmd.Flags().StringVarP(&configBucket, "bucket", "b", "", "Specify your bucket")
	_ = configCmd.MarkFlagRequired("bucket")

	configCmd.Flags().StringVarP(&configRegion, "region", "r", "", "Specify your region")
	configCmd.Flags().StringVarP(&configEndpoint, "endpoint", "e", "", "Specify COS endpoint")

	configCmd.Flags().IntVarP(&configMaxThread, "max_thread", "m", 5, "Specify the number of threads")
	configCmd.Flags().IntVarP(&configPartSize, "part_size", "p", 20, "Specify min part size in MB")
	configCmd.Flags().IntVar(&configRetryTimes, "retry", 5, "Specify retry times")
	configCmd.Flags().IntVar(&configTimeout, "timeout", 60, "Specify request timeout")
	configCmd.Flags().StringVarP(&configAppId, "appid", "u", "", "Specify your appid")
	configCmd.Flags().StringVar(&configVerifyMethod, "verify", "md5", "Specify your encryption method")
	configCmd.Flags().BoolVar(&configNoSsl, "do-not-use-ssl", false, "Use http://")
	configCmd.Flags().BoolVar(&configAnonymous, "anonymous", false, "Anonymous operation")
}

// Save config
func config(cmd *cobra.Command, _ []string) error {
	cmd.Flags().Visit(func(flag *pflag.Flag) {
		log.Debugf("%s: %v", flag.Name, flag.Value)
	})
	cfg := ini.Empty()
	commonSection, err := cfg.NewSection("common")
	if err != nil {
		log.Error("Cannot create section `common`")
		return Error{
			Code:    -1,
			Message: "cannot create section 'common'",
		}
	}
	newKey(commonSection, "secret_id", configSecretId)
	newKey(commonSection, "secret_key", configSecretKey)
	if configToken != "" {
		newKey(commonSection, "token", configToken)
	}
	newKey(commonSection, "bucket", configBucket)
	if configRegion == "" && configEndpoint == "" {
		err = fmt.Errorf("error: one of the arguments -r/--region -e/--endpoint is required")
		return Error{
			Code:    1,
			Message: err.Error(),
		}
	} else if configRegion != "" {
		newKey(commonSection, "region", configRegion)
	} else {
		newKey(commonSection, "endpoint", configEndpoint)
	}
	newKey(commonSection, "max_thread", strconv.Itoa(configMaxThread))
	newKey(commonSection, "part_size", strconv.Itoa(configPartSize))
	newKey(commonSection, "retry", strconv.Itoa(configRetryTimes))
	newKey(commonSection, "timeout", strconv.Itoa(configTimeout))
	if configAppId != "" {
		newKey(commonSection, "appid", configAppId)
	}
	if configNoSsl {
		newKey(commonSection, "schema", "http")
	} else {
		newKey(commonSection, "schema", "https")
	}
	newKey(commonSection, "verify", configVerifyMethod)
	if configAnonymous {
		newKey(commonSection, "anonymous", "True")
	} else {
		newKey(commonSection, "anonymous", "False")
	}
	err = cfg.SaveTo(cli.ConfigPath)
	if err != nil {
		log.Errorf("Cannot write file to %s", cli.ConfigPath)
		return Error{
			Code:    -1,
			Message: fmt.Sprintf("cannot write file to %s", cli.ConfigPath),
		}
	}
	log.Infof("Created configuration file in %s", cli.ConfigPath)
	return nil
}

func newKey(section *ini.Section, name, val string) {
	_, err := section.NewKey(name, val)
	if err != nil {
		panic(err.Error())
	}
}
