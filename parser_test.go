package yarnlock

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"testing"
)

//go:embed yarn.lock
var y string

func TestName(t *testing.T) {
	tokenizer := new(_Tokenizer)
	if e := tokenizer.tokenize(y); e != nil {
		panic(e)
	}
	parser := _Parser{
		fileLoc:  "yarn.lock",
		token:    tokenizer.tokens[0],
		tokens:   tokenizer.tokens,
		tokenPtr: 0,
		comments: nil,
	}
	parser.next()
	//fmt.Println(parser.parse(0))
	b, e := json.MarshalIndent(parser.parse(0), "", "  ")
	if e != nil {
		panic(e)
	}
	fmt.Println(string(b))
}

func TestName2(t *testing.T) {
	r, e := ParseLockFileData([]byte(y))
	t.Log(r, e)
	if e != nil {
		panic(e)
	}
}
