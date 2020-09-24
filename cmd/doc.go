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

/*
	Package cmd implements all commands in cosutil. In this package, each
	subcommand is a single go file, and root is the cosutil command itself.

	Due to the restriction of dependency cobra, the global cosutil command
	arguments must be between the `cosutil` and the subcommand. For example,
	if you want to enable debug mode when downloading, you must use

		cosutil -d download cospath localpath

	but not

		cosutil download -d cospath localpath

	which will cause an parse error.
*/
package cmd
