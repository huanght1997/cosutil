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

package cli

import (
	"context"

	log "github.com/sirupsen/logrus"
)

func (client *Client) GetBucketVersioning() bool {
	result, _, err := client.Client.Bucket.GetVersioning(context.Background())
	if err != nil {
		log.Warn(err.Error())
		return false
	} else {
		if result.Status == "" {
			log.Info("Not configured")
		} else {
			log.Info(result.Status)
		}
		return true
	}
}
