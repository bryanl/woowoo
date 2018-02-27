// Copyright 2017 The ksonnet authors
//
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package strings

import (
	"bytes"
	"fmt"
	"path"
	"strings"

	"github.com/PuerkitoBio/purell"
)

// IsASCIIIdentifier takes a string and returns true if the string does not
// contain any special characters.
func IsASCIIIdentifier(s string) bool {
	f := func(r rune) bool {
		return r < 'A' || r > 'z'
	}
	if strings.IndexFunc(s, f) != -1 {
		return false
	}
	return true
}

// QuoteNonASCII puts quotes around an identifier that contains non-ASCII
// characters.
func QuoteNonASCII(s string) string {
	if !IsASCIIIdentifier(s) {
		return fmt.Sprintf(`"%s"`, s)
	}
	return s
}

// NormalizeURL uses purell's "usually safe normalization" algorithm to
// normalize URLs. This includes removing dot segments, removing trailing
// slashes, removing unnecessary escapes, removing default ports, and setting
// the URL to lowercase.
func NormalizeURL(s string) (string, error) {
	return purell.NormalizeURLString(s, purell.FlagsUsuallySafeGreedy)
}

func PadRows(rows [][]string) (string, error) {
	maxRowLen := 0
	for _, row := range rows {
		if rowLen := len(row); rowLen > maxRowLen {
			maxRowLen = rowLen
		}
	}

	colMaxes := make([]int, maxRowLen)
	for currCol := 0; currCol < maxRowLen; currCol++ {
		for _, row := range rows {
			rowLen := len(row)
			if currCol >= rowLen {
				continue
			}

			cellLen := len(row[currCol])
			if currCol < rowLen && colMaxes[currCol] < cellLen {
				colMaxes[currCol] = cellLen
			}
		}
	}

	var err error
	var buf bytes.Buffer
	for _, row := range rows {
		rowLen := len(row)
		for j, col := range row {
			_, err = buf.WriteString(col)
			if err != nil {
				return "", err
			}

			// Don't add space to the end of the last column.
			if j >= rowLen-1 {
				continue
			}

			padSize := colMaxes[j] + 1 - len(col)
			_, err = buf.WriteString(strings.Repeat(" ", padSize))
			if err != nil {
				return "", err
			}
		}

		// Add a newline to the end of the row (but only if there is more
		// than 0 rows).
		_, err = buf.WriteString("\n")
		if err != nil {
			return "", err
		}
	}

	return buf.String(), nil
}

// AppendToPath appends one or more paths to the specified original path.
func AppendToPath(originalPath string, toAppend ...string) string {
	paths := append([]string{originalPath}, toAppend...)
	return path.Join(paths...)
}
