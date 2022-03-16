package yarnlock

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"regexp"
	"strconv"
	"strings"
)

type ValueType int

const (
	TokenValueVoid ValueType = iota
	TokenValueInt
	TokenValueString
	TokenValueBool
)

type TokenType int

const (
	TokenBoolean TokenType = iota + 1
	TokenString
	TokenIdentifier
	TokenEOF
	TokenColon
	TokenNewLine
	TokenComment
	TokenIndent
	TokenInvalid
	TokenNumber
	TokenComma
)

type Token struct {
	Line  int
	Col   int
	Type  TokenType
	Value TokenValue
}

type TokenValue struct {
	Int       int
	String    string
	Bool      bool
	Valid     bool
	ValueType ValueType
}

func (t TokenValue) MarshalText() ([]byte, error) {
	switch t.ValueType {
	case TokenValueInt:
		return []byte(strconv.Itoa(t.Int)), nil
	case TokenValueBool:
		return []byte(strconv.FormatBool(t.Bool)), nil
	case TokenValueString:
		return []byte(t.String), nil
	default:
		return []byte("void"), nil
	}
}

func (t *TokenValue) IsEmpty() bool {
	return !t.Valid
}

func (t *Token) isString() bool {
	return t.Value.ValueType == TokenValueString
}

type Tokenizer struct {
	lastNewLine bool
	line        int
	col         int
	tokens      []Token
}

func (t *Tokenizer) buildToken(tt TokenType, value interface{}) {
	tk := Token{
		Line:  t.line,
		Col:   t.col,
		Type:  tt,
		Value: TokenValue{},
	}
	if tt == TokenComment || tt == TokenString {
		tk.Value = TokenValue{String: value.(string), ValueType: TokenValueString}
	} else if tt == TokenBoolean {
		tk.Value = TokenValue{Bool: value.(bool), ValueType: TokenValueBool}
	} else if tt == TokenNumber || tt == TokenIndent {
		tk.Value = TokenValue{Int: value.(int), ValueType: TokenValueInt}
	}
	tk.Value.Valid = true
	if tt == TokenInvalid {
		panic(1)
	}
	t.tokens = append(t.tokens, tk)
}

func (t *Tokenizer) tokenize(input string) error {
	for len(input) > 0 {
		var chop = 0
		if input[0] == '\n' || input[0] == '\r' {
			chop++
			if len(input) > 1 && input[1] == '\n' {
				chop++
			}
			t.line++
			t.col = 0
			t.buildToken(TokenNewLine, nil)
		} else if input[0] == '#' {
			chop++
			var nextNewLine = strings.Index(input[chop:], "\n")
			if nextNewLine == -1 {
				nextNewLine = len(input)
			}
			nextNewLine += chop // workaround for go haven't IndexN
			val := input[chop:nextNewLine]
			chop = nextNewLine
			t.buildToken(TokenComment, val)
		} else if input[0] == ' ' {
			if t.lastNewLine {
				indentSize := 1
				for i := 1; input[i] == ' '; i++ {
					indentSize++
				}
				if indentSize%2 == 1 {
					return errors.New("Invalid number of spaces")
				} else {
					chop = indentSize
					t.buildToken(TokenIndent, indentSize/2)
				}
			} else {
				chop++
			}
		} else if input[0] == '"' {
			i := 1
			for ; i < len(input); i++ {
				if input[i] == '"' {
					isEscaped := input[i-1] == '\\' && input[i-2] != '\\'
					if !isEscaped {
						i++
						break
					}
				}
			}
			val := input[:i]
			chop = i
			var s string
			if e := json.Unmarshal([]byte(val), &s); e != nil {
				t.buildToken(TokenInvalid, nil)
			} else {
				t.buildToken(TokenString, s)
			}
		} else if input[0] >= '0' && input[0] <= '9' {
			val := _numberPattern.FindString(input)
			chop = len(val)
			n, _ := strconv.Atoi(val)
			t.buildToken(TokenNumber, n)
		} else if strings.HasPrefix(input, "true") {
			t.buildToken(TokenBoolean, true)
			chop = 4
		} else if strings.HasSuffix(input, "false") {
			t.buildToken(TokenBoolean, false)
			chop = 5
		} else if input[0] == ':' {
			t.buildToken(TokenColon, nil)
			chop++
		} else if input[0] == ',' {
			t.buildToken(TokenComma, nil)
			chop++
		} else if _strPattern.MatchString(input) {
			i := 0
			for ; i < len(input); i++ {
				char := input[i]
				if char == ':' || char == ' ' || char == '\r' || char == '\n' || char == ',' {
					break
				}
			}
			name := input[:i]
			chop = i
			t.buildToken(TokenString, name)
		} else {
			t.buildToken(TokenInvalid, nil)
		}
		if chop == 0 {
			t.buildToken(TokenInvalid, nil)
		}
		t.col += chop
		t.lastNewLine = input[0] == '\n' || (input[0] == '\r' && input[1] == '\n')
		if chop == 0 {
			panic("chop is zero")
		}
		input = input[chop:]
	}
	t.buildToken(TokenEOF, nil)
	return nil
}

var _numberPattern = regexp.MustCompile("^\\d+")
var _strPattern = regexp.MustCompile("^[a-zA-Z\\/.-]")
var _versionRegex = regexp.MustCompile("^yarn lockfile v(\\d+)$")

const LockfileVersion = 1

type Parser struct {
	fileLoc  string
	token    Token
	tokens   []Token
	tokenPtr int
	comments []string
}

func (p *Parser) onComment(token Token) {
	if !token.isString() {
		panic("expected token value to be a string")
	}
	comment := strings.TrimSpace(token.Value.String)

	versionMatch := _versionRegex.FindStringSubmatch(comment)
	if len(versionMatch) > 0 {
		version, _ := strconv.Atoi(versionMatch[1])
		if version > LockfileVersion {
			panic(fmt.Sprintf("Can't install from a lockfile of version %d as you're on an old yarn version that only supports versions up to %d. Run \\`$ yarn self-update\\` to upgrade to the latest version.", version, LockfileVersion))
		}
	}
	p.comments = append(p.comments, comment)
}

func (p *Parser) next() Token {
	if p.tokenPtr >= len(p.tokens) {
		panic("No more tokens")
	}
	tk := p.tokens[p.tokenPtr]
	p.tokenPtr++
	if tk.Type == TokenComment {
		p.onComment(tk)
		return p.next()
	} else {
		p.token = tk
		return tk
	}
}

func (p *Parser) unexpected(msg string) {
	if msg == "" {
		panic("Unexpected token")
	} else {
		panic(fmt.Sprintf("%s%d:%d in %s", msg, p.token.Line, p.token.Col, p.fileLoc))
	}
}

func (p *Parser) expect(tt TokenType) {
	if p.token.Type == tt {
		p.next()
	} else {
		p.unexpected("")
	}
}

func (p *Parser) eat(tt TokenType) bool {
	if p.token.Type == tt {
		p.next()
		return true
	} else {
		return false
	}
}

func (p *Parser) parse(indent int) interface{} {
	obj := map[TokenValue]interface{}{}
	for {
		propToken := p.token
		if propToken.Type == TokenNewLine {
			nextToken := p.next()
			if indent == 0 {
				// if we have 0 indentation then the next token doesn't matter
				continue
			}
			if nextToken.Type != TokenIndent {
				// if we have no indentation after a newline then we've gone down a level
				break
			}
			if nextToken.Value.Int == indent {
				// all is good, the indent is on our level
				p.next()
			} else {
				// the indentation is less than our level
				break
			}
		} else if propToken.Type == TokenIndent {
			if propToken.Value.Int == indent {
				p.next()
			} else {
				break
			}
		} else if propToken.Type == TokenEOF {
			break
		} else if propToken.Type == TokenString {
			// property key
			key := propToken.Value
			if key.IsEmpty() {
				panic("Expected a key")
			}
			keys := []TokenValue{key}
			p.next()
			// support multiple keys
			for p.token.Type == TokenComma {
				p.next() // skip comma

				keyToken := p.token
				if keyToken.Type != TokenString {
					p.unexpected("Expected string")
				}

				key := keyToken.Value
				if key.IsEmpty() {
					panic("Expected a key")
				}
				keys = append(keys, key)
				p.next()
			}
			wasColon := p.token.Type == TokenColon
			if wasColon {
				p.next()
			}
			if isValidPropValueToken(p.token) {
				for _, key := range keys {
					obj[key] = p.token.Value // 299
				}
				p.next()
			} else if wasColon {
				val := p.parse(indent + 1)
				for _, key := range keys {
					obj[key] = val
				}
				if indent != 0 && p.token.Type != TokenIndent {
					break
				}
			} else {
				p.unexpected("Invalid value type")
			}
		} else {
			p.unexpected(fmt.Sprintf("Unknown token: %v", propToken))
		}
	}
	return obj
}

func isValidPropValueToken(token Token) bool {
	return token.Type == TokenBoolean || token.Type == TokenString || token.Type == TokenNumber
}
