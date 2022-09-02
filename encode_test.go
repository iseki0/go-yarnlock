package yarnlock

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapping(t *testing.T) {
	type Case struct {
		input    string
		expected string
	}
	cases := []Case{
		{`1.2.3`, `"1.2.3"`},
		{`^1.2.3`, `"^1.2.3"`},
		{`@foo/bar`, `"@foo/bar"`},
		{`true`, `"true"`},
		{`false`, `"false"`},
		{`https://foo.org`, `"https://foo.org"`},
		{`foo`, `foo`},
		{`sha512-JIB2+XJrb7v3zceV2XzDhGIB902CmKGSpSl4q2C6agU9SNLG/2V1RtFRGPG1Ajh9STj3+q6zJMOC+N/pp2P9DA==`, `sha512-JIB2+XJrb7v3zceV2XzDhGIB902CmKGSpSl4q2C6agU9SNLG/2V1RtFRGPG1Ajh9STj3+q6zJMOC+N/pp2P9DA==`},
		{`>=2.2.7 <3`, `">=2.2.7 <3"`},
	}

	for _, testCase := range cases {
		assert.Equal(t, testCase.expected, maybeWrap(testCase.input))
	}
}

func TestEncodeMap(t *testing.T) {
	type Case struct {
		m        map[string]string
		expected []string
	}
	cases := []Case{
		{
			m: map[string]string{
				"foo":      "1.2.3",
				"true":     "1.2.3",
				"@foo/bar": "1.2.3",
			},
			expected: []string{`test:`,
				`  "@foo/bar" "1.2.3"`,
				`  "true" "1.2.3"`,
				`  foo "1.2.3"`,
			},
		},
	}

	for _, testCase := range cases {
		assert.Equal(t, testCase.expected, encodeMap(testCase.m, "test", ""))
	}
}

func TestRoundtrip(t *testing.T) {
	r, e := ParseLockFileData([]byte(y))
	if e != nil {
		t.Error(e)
	}
	var b bytes.Buffer
	if e = r.Encode(&b); e != nil {
		t.Error(e)
	}

	assert.Equal(t, y, b.String())
}
