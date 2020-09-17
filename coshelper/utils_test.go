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
package coshelper

import (
	"testing"
)

func TestGetFileMd5(t *testing.T) {
	actual := GetFileMd5("../LICENSE")
	expected := "3B83EF96387F14655FC854DDC3C6BD57"
	if actual != expected {
		t.Errorf("Get MD5 string: %s, expected: %s", actual, expected)
	}
}

func TestConvertStringToHeader(t *testing.T) {
	headerString := `{'x-cos-storage-class':'Archive',
    'content-length':114514,
    'x-cos-meta-example':'The McDonald\'s said:\"Sorry, there are no French fries.\"'
}`
	header := ConvertStringToHeader(headerString)
	header.Set("X-Cos-Meta-Test", "Hello!")
	cases := []struct {
		input, expected string
	}{
		{"x-cos-storage-class", "Archive"},
		{"content-length", "114514"},
		{"x-cos-meta-example", `The McDonald's said:"Sorry, there are no French fries."`},
		{"x-cos-meta-test", "Hello!"},
		{"non-exist-header", ""},
	}
	for _, c := range cases {
		actual := header.Get(c.input)
		if actual != c.expected {
			t.Errorf("header[%s]=%s, expected %s",
				c.input, actual, c.expected)
		}
	}
}
