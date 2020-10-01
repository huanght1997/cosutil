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

func (client *Client) DeleteBucket(force bool) bool {
	if force {
		log.Info("Clearing files and upload parts in the bucket")
		client.AbortParts("")
		client.DeleteFolder("", &DeleteOption{
			Force:    true,
			Versions: false,
		})
		client.DeleteFolder("", &DeleteOption{
			Force:    true,
			Versions: true,
		})
	}
	_, err := client.Client.Bucket.Delete(context.Background())
	if err != nil {
		log.Warn(err.Error())
		return false
	}
	log.Infof("Delete cos://%s", client.Config.Bucket)
	return true
}
