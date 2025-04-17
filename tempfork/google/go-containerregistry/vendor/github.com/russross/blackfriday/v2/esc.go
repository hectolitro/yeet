// Copyright 2025 AUTHORS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package blackfriday

import (
	"html"
	"io"
)

var htmlEscaper = [256][]byte{
	'&': []byte("&amp;"),
	'<': []byte("&lt;"),
	'>': []byte("&gt;"),
	'"': []byte("&quot;"),
}

func escapeHTML(w io.Writer, s []byte) {
	escapeEntities(w, s, false)
}

func escapeAllHTML(w io.Writer, s []byte) {
	escapeEntities(w, s, true)
}

func escapeEntities(w io.Writer, s []byte, escapeValidEntities bool) {
	var start, end int
	for end < len(s) {
		escSeq := htmlEscaper[s[end]]
		if escSeq != nil {
			isEntity, entityEnd := nodeIsEntity(s, end)
			if isEntity && !escapeValidEntities {
				w.Write(s[start : entityEnd+1])
				start = entityEnd + 1
			} else {
				w.Write(s[start:end])
				w.Write(escSeq)
				start = end + 1
			}
		}
		end++
	}
	if start < len(s) && end <= len(s) {
		w.Write(s[start:end])
	}
}

func nodeIsEntity(s []byte, end int) (isEntity bool, endEntityPos int) {
	isEntity = false
	endEntityPos = end + 1

	if s[end] == '&' {
		for endEntityPos < len(s) {
			if s[endEntityPos] == ';' {
				if entities[string(s[end:endEntityPos+1])] {
					isEntity = true
					break
				}
			}
			if !isalnum(s[endEntityPos]) && s[endEntityPos] != '&' && s[endEntityPos] != '#' {
				break
			}
			endEntityPos++
		}
	}

	return isEntity, endEntityPos
}

func escLink(w io.Writer, text []byte) {
	unesc := html.UnescapeString(string(text))
	escapeHTML(w, []byte(unesc))
}
