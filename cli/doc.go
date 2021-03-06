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

// Package cli implements functions used to call the API in Tencent Cloud
// COS(Cloud Object Storage). These functions are all in a struct Client.
//
// If you want to use the functions in this package, you should first fill
// the field of struct ClientConfig, and call NewClient to generate one
// Client, and use this client to call the functions you need.
package cli
