package gocqlmem

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"regexp"
	"slices"
	"strings"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/capillariesio/capillaries/pkg/eval"
	"golang.org/x/tools/go/ast/astutil"
)

type LexemType int

const (
	LexemStringLiteral LexemType = iota
	LexemNumberLiteral
	LexemBoolLiteral
	LexemIdent
	LexemPointedIdent
	LexemKeyword
	LexemComma
	LexemSemicolon
	LexemArithmeticOp
	LexemLogicalOp
	LexemCqlOp
	LexemParenthesis
	LexemAsterisk
	LexemPointedAsterisk
	LexemNull
	LexemQuestionMark
	LexemAs
)

type Lexem struct {
	T LexemType
	V string
}

type ClusteringOrderType int

const (
	ClusteringOrderNone ClusteringOrderType = iota
	ClusteringOrderAsc
	ClusteringOrderDesc
)

func stringToClusteringOrderType(s string) ClusteringOrderType {
	switch s {
	case "ASC":
		return ClusteringOrderAsc
	case "DESC":
		return ClusteringOrderDesc
	default:
		return ClusteringOrderNone
	}
}

type OrderByField struct {
	FieldName       string
	ClusteringOrder ClusteringOrderType
}

type Command interface {
	GetCtxKeyspace() string
	SetCtxKeyspace(string)
}

type KeyValuePair struct {
	K string
	V *Lexem
}

type CommandCreateKeyspace struct {
	IfNotExists     bool
	KeyspaceName    string
	WithReplication []*KeyValuePair
}

func (c *CommandCreateKeyspace) GetCtxKeyspace() string {
	return c.KeyspaceName
}
func (c *CommandCreateKeyspace) SetCtxKeyspace(keyspace string) {
}

type CommandUseKeyspace struct {
	KeyspaceName string
}

func (c *CommandUseKeyspace) GetCtxKeyspace() string {
	return c.KeyspaceName
}
func (c *CommandUseKeyspace) SetCtxKeyspace(keyspace string) {
}

type CommandDropKeyspace struct {
	IfExists     bool
	KeyspaceName string
}

func (c *CommandDropKeyspace) GetCtxKeyspace() string {
	return c.KeyspaceName
}
func (c *CommandDropKeyspace) SetCtxKeyspace(keyspace string) {
}

type CreateTableColumnDef struct {
	Name       string
	ColumnType gocql.Type
}

type ColumnSetExp struct {
	Name      string
	ExpLexems []*Lexem
}

type CommandCreateTable struct {
	CtxKeyspace          string
	IfNotExists          bool
	TableName            string
	ColumnDefs           []*CreateTableColumnDef
	PartitionKeyColumns  []string
	ClusteringKeyColumns []string
	ClusteringOrderBy    []*OrderByField
}

func (c *CommandCreateTable) GetCtxKeyspace() string {
	return c.CtxKeyspace
}
func (c *CommandCreateTable) SetCtxKeyspace(keyspace string) {
	c.CtxKeyspace = keyspace
}

type CommandTruncateTable struct {
	CtxKeyspace string
	TableName   string
}

func (c *CommandTruncateTable) GetCtxKeyspace() string {
	return c.CtxKeyspace
}
func (c *CommandTruncateTable) SetCtxKeyspace(keyspace string) {
	c.CtxKeyspace = keyspace
}

type CommandDropTable struct {
	CtxKeyspace string
	IfExists    bool
	TableName   string
}

func (c *CommandDropTable) GetCtxKeyspace() string {
	return c.CtxKeyspace
}
func (c *CommandDropTable) SetCtxKeyspace(keyspace string) {
	c.CtxKeyspace = keyspace
}

// ORDER BY: The partition key must be defined in the WHERE clause and then the ORDER BY clause defines
// one or more clustering columns to use for ordering. The order of the specified columns must match the
// order of the clustering columns in the PRIMARY KEY definition
type CommandSelect struct {
	CtxKeyspace     string
	Distinct        bool
	SelectExpLexems [][]*Lexem
	TableName       string
	WhereExpLexems  []*Lexem
	OrderByFields   []*OrderByField
	Limit           *Lexem
	SelectExpAsts   []ast.Expr
	SelectExpNames  []string
	WhereExpAst     ast.Expr
}

func (c *CommandSelect) GetCtxKeyspace() string {
	return c.CtxKeyspace
}
func (c *CommandSelect) SetCtxKeyspace(keyspace string) {
	c.CtxKeyspace = keyspace
}

type CommandInsert struct {
	CtxKeyspace string
	TableName   string

	ColumnNames []string

	ColumnValueLexems [][]*Lexem
	//ColumnValueExpAsts []ast.Expr
	ColumnValues []any

	IfNotExists bool
}

func (c *CommandInsert) GetCtxKeyspace() string {
	return c.CtxKeyspace
}
func (c *CommandInsert) SetCtxKeyspace(keyspace string) {
	c.CtxKeyspace = keyspace
}

type CommandUpdate struct {
	CtxKeyspace string
	TableName   string

	ColumnSetExpressions []*ColumnSetExp
	ColumnSetExpAsts     []ast.Expr

	WhereExpLexems []*Lexem
	WhereExpAst    ast.Expr

	IfExists    bool
	IfExpLexems []*Lexem
	IfExpAst    ast.Expr
}

func (c *CommandUpdate) GetCtxKeyspace() string {
	return c.CtxKeyspace
}
func (c *CommandUpdate) SetCtxKeyspace(keyspace string) {
	c.CtxKeyspace = keyspace
}

type CommandDelete struct {
	CtxKeyspace     string
	TableName       string
	ColumnsToDelete []string
	WhereExpLexems  []*Lexem
	IfExists        bool
	WhereExpAst     ast.Expr
}

func (c *CommandDelete) GetCtxKeyspace() string {
	return c.CtxKeyspace
}
func (c *CommandDelete) SetCtxKeyspace(keyspace string) {
	c.CtxKeyspace = keyspace
}

func skipBlank(s string) string {
	return strings.TrimLeft(s, " \t\r\n")
}

func getStringLiteral(s string) (*Lexem, string) {
	s = skipBlank(s)
	r := regexp.MustCompile(`^'(''|[^'])*'`)
	litRange := r.FindStringIndex(s)
	if len(litRange) >= 2 {
		return &Lexem{LexemStringLiteral, s[1 : litRange[1]-1]}, s[litRange[1]:]
	}
	return nil, s
}

func getNull(s string) (*Lexem, string) {
	s = skipBlank(s)
	r := regexp.MustCompile(`^(?i)NULL\b`)
	litRange := r.FindStringIndex(s)
	if len(litRange) >= 2 {
		return &Lexem{LexemNull, s[0:litRange[1]]}, s[litRange[1]:]
	}
	return nil, s
}

func getQuestionMark(s string) (*Lexem, string) {
	s = skipBlank(s)
	r := regexp.MustCompile(`^(?i)\?`)
	litRange := r.FindStringIndex(s)
	if len(litRange) >= 2 {
		return &Lexem{LexemQuestionMark, s[0:litRange[1]]}, s[litRange[1]:]
	}
	return nil, s
}

func getAs(s string) (*Lexem, string) {
	s = skipBlank(s)
	r := regexp.MustCompile(`^(?i)AS\b`)
	litRange := r.FindStringIndex(s)
	if len(litRange) >= 2 {
		return &Lexem{LexemAs, strings.ToUpper(s[0:litRange[1]])}, s[litRange[1]:]
	}
	return nil, s
}

func getNumberLiteral(s string) (*Lexem, string) {
	s = skipBlank(s)
	r := regexp.MustCompile(`^[\-\+]?\d*\.?\d+([eE][-+]?\d+)?`)
	litRange := r.FindStringIndex(s)
	if len(litRange) >= 2 {
		return &Lexem{LexemNumberLiteral, s[0:litRange[1]]}, s[litRange[1]:]
	}
	return nil, s
}

func getBoolLiteral(s string) (*Lexem, string) {
	s = skipBlank(s)
	r := regexp.MustCompile(`^(?i)(TRUE|FALSE)(\b)`)
	litRange := r.FindStringIndex(s)
	if len(litRange) >= 2 {
		return &Lexem{LexemBoolLiteral, strings.ToUpper(s[0:litRange[1]])}, s[litRange[1]:]
	}
	return nil, s
}

func getKeyword(s string, kwRegex string, isProcess bool) (*Lexem, string) {
	s = skipBlank(s)
	r := regexp.MustCompile(`^` + kwRegex)
	litRange := r.FindStringIndex(s)
	if len(litRange) >= 2 {
		reBlank := regexp.MustCompile(`\s+`)
		result := strings.ToUpper(reBlank.ReplaceAllString(s[0:litRange[1]], " "))
		if isProcess {
			return &Lexem{LexemKeyword, result}, s[litRange[1]:]
		} else {
			return &Lexem{LexemKeyword, result}, s
		}
	}
	return nil, s
}

func getIdent(s string) (*Lexem, string) {
	s = skipBlank(s)
	r := regexp.MustCompile(`^("[a-zA-Z][a-zA-Z0-9_]*"|[a-zA-Z][a-zA-Z0-9_]*)`)
	litRange := r.FindStringIndex(s)
	if len(litRange) >= 2 {
		return &Lexem{LexemIdent, strings.ReplaceAll(s[0:litRange[1]], `"`, ``)}, s[litRange[1]:]
	}
	return nil, s
}

func getPointedIdent(s string) (*Lexem, string) {
	s = skipBlank(s)
	r := regexp.MustCompile(`^("[a-zA-Z][a-zA-Z0-9_]*"|[a-zA-Z][a-zA-Z0-9_]*)\.([a-zA-Z][a-zA-Z0-9_]*"|[a-zA-Z][a-zA-Z0-9_]*)`)
	litRange := r.FindStringIndex(s)
	if len(litRange) >= 2 {
		return &Lexem{LexemPointedIdent, strings.ReplaceAll(s[0:litRange[1]], `"`, ``)}, s[litRange[1]:]
	}
	return nil, s
}

func getIdentOrPointedIdent(s string) (*Lexem, string) {
	var l *Lexem
	if l, s = getPointedIdent(s); l != nil {
		return l, s
	}
	return getIdent(s)
}

func getArithmeticOp(s string) (*Lexem, string) {
	s = skipBlank(s)
	r := regexp.MustCompile(`^[\+\-*/%]`)
	litRange := r.FindStringIndex(s)
	if len(litRange) >= 2 {
		return &Lexem{LexemArithmeticOp, s[0:litRange[1]]}, s[litRange[1]:]
	}
	return nil, s
}

func getLogicalOp(s string) (*Lexem, string) {
	s = skipBlank(s)
	r := regexp.MustCompile(`^(?i)(OR\b|AND\b|=|!=|>=|<=|>|<)`)
	litRange := r.FindStringIndex(s)
	if len(litRange) >= 2 {
		opValue := strings.ToUpper(s[0:litRange[1]])
		switch opValue {
		case "=":
			opValue = "=="
		case "AND":
			opValue = "&&"
		case "OR":
			opValue = "||"
		}
		return &Lexem{LexemLogicalOp, opValue}, s[litRange[1]:]
	}
	return nil, s
}

func getParenthesis(s string, isProcess bool) (*Lexem, string) {
	s = skipBlank(s)
	r := regexp.MustCompile(`^[()]`)
	litRange := r.FindStringIndex(s)
	if len(litRange) >= 2 {
		reBlank := regexp.MustCompile(`\s+`)
		result := strings.ToUpper(reBlank.ReplaceAllString(s[0:litRange[1]], " "))
		if isProcess {
			return &Lexem{LexemParenthesis, result}, s[litRange[1]:]
		} else {
			return &Lexem{LexemParenthesis, result}, s
		}
	}

	return nil, s
}

func getAsterisk(s string) (*Lexem, string) {
	s = skipBlank(s)
	r := regexp.MustCompile(`^\*`)
	litRange := r.FindStringIndex(s)
	if len(litRange) >= 2 {
		return &Lexem{LexemAsterisk, s[0:litRange[1]]}, s[litRange[1]:]
	}
	return nil, s
}

func getAsteriskOrPointedAsterisk(s string) (*Lexem, string) {
	var l *Lexem
	if l, s = getPointedAsterisk(s); l != nil {
		return l, s
	}
	return getAsterisk(s)
}

func getPointedAsterisk(s string) (*Lexem, string) {
	s = skipBlank(s)
	r := regexp.MustCompile(`^("[a-zA-Z][a-zA-Z0-9_]*"|[a-zA-Z][a-zA-Z0-9_]*)\.(\*)`)
	litRange := r.FindStringIndex(s)
	if len(litRange) >= 2 {
		return &Lexem{LexemPointedAsterisk, strings.ReplaceAll(s[0:litRange[1]], `"`, ``)}, s[litRange[1]:]
	}
	return nil, s
}

func getCqlOp(s string) (*Lexem, string) {
	s = skipBlank(s)
	r := regexp.MustCompile(`^(?i)(NOT\s+IN\b|IN\b|!~|~)(\b)`)
	litRange := r.FindStringIndex(s)
	if len(litRange) >= 2 {
		return &Lexem{LexemCqlOp, strings.ToUpper(s[0:litRange[1]])}, s[litRange[1]:]
	}
	return nil, s
}

func getComma(s string) (*Lexem, string) {
	s = skipBlank(s)
	r := regexp.MustCompile(`^,`)
	litRange := r.FindStringIndex(s)
	if len(litRange) >= 2 {
		return &Lexem{LexemComma, s[0:litRange[1]]}, s[litRange[1]:]
	}
	return nil, s
}

func getSemicolon(s string) (*Lexem, string) {
	s = skipBlank(s)
	r := regexp.MustCompile(`^;`)
	litRange := r.FindStringIndex(s)
	if len(litRange) >= 2 {
		return &Lexem{LexemSemicolon, s[0:litRange[1]]}, s[litRange[1]:]
	}
	return nil, s
}

func getSelectExpressionLexems(s string) ([]*Lexem, string) {
	var l *Lexem
	parenthesisStackLen := 0
	lexems := make([]*Lexem, 0)
	for {
		stopWord := `(?i)FROM\b`
		l, s = getKeyword(s, stopWord, false)
		if l != nil {
			if parenthesisStackLen == 0 {
				break
			}
			l, s = getKeyword(s, stopWord, true)
			lexems = append(lexems, l)
			continue
		}
		// Stop comma - swallow if it's within parenthesis
		if l, s = getComma(s); l != nil {
			if parenthesisStackLen == 0 {
				break
			}
			lexems = append(lexems, l)
			continue
		}
		// Parenthesis
		if l, s = getParenthesis(s, true); l != nil {
			if l.V == "(" {
				parenthesisStackLen++
			} else {
				parenthesisStackLen--
			}
			lexems = append(lexems, l)
			continue
		}
		if l, s = getQuestionMark(s); l != nil {
			lexems = append(lexems, l)
			continue
		}
		if len(lexems) == 0 || lexems[len(lexems)-1].V == "(" || lexems[len(lexems)-1].T == LexemComma || lexems[len(lexems)-1].T == LexemArithmeticOp || lexems[len(lexems)-1].T == LexemLogicalOp {
			// No arithmetic op allowed here
			if l, s = getBoolLiteral(s); l != nil {
				lexems = append(lexems, l)
				continue
			}
			if l, s = getNull(s); l != nil {
				lexems = append(lexems, l)
				continue
			}
			if l, s = getStringLiteral(s); l != nil {
				lexems = append(lexems, l)
				continue
			}
			if l, s = getNumberLiteral(s); l != nil {
				lexems = append(lexems, l)
				continue
			}
		} else {
			// No literal allowed here
			if l, s = getArithmeticOp(s); l != nil {
				lexems = append(lexems, l)
				continue
			}
			if l, s = getLogicalOp(s); l != nil {
				lexems = append(lexems, l)
				continue
			}
		}
		if l, s = getAs(s); l != nil {
			lexems = append(lexems, l)
			continue
		}
		if l, s = getAsteriskOrPointedAsterisk(s); l != nil {
			lexems = append(lexems, l)
			continue
		}
		if l, s = getCqlOp(s); l != nil {
			lexems = append(lexems, l)
			continue
		}
		if l, s = getIdentOrPointedIdent(s); l != nil {
			lexems = append(lexems, l)
			continue
		}
		break
	}

	return lexems, s
}

func getSelectExpressions(s string) ([][]*Lexem, string) {
	var l *Lexem
	exps := make([][]*Lexem, 0)
	for {
		l, s = getComma(s)
		if l != nil {
			continue
		}
		var exp []*Lexem
		exp, s = getSelectExpressionLexems(s)
		if len(exp) == 0 {
			break
		}
		exp = convertCastForAstParser(exp)
		exps = append(exps, exp)
	}

	return exps, s
}

// Returns indices of the first found "Ident,IN,(1,2,3)" "ident,IN,?" or lexem sequence
func findInNotInLexem(lexems []*Lexem) (int, int) {
	for i := range len(lexems) {
		if lexems[i].T == LexemCqlOp && (lexems[i].V == "IN" || lexems[i].V == "NOT IN") && i > 0 && lexems[i-1].T == LexemIdent {
			if i < len(lexems)-2 && lexems[i+1].V == "(" {
				// It's "IN (1,2,3)"
				startIdx := i - 1 // Result starts at the ident
				endIdx := i + 2   // Start with the first arg of the IN/NOT IN sequence
				for endIdx < len(lexems) {
					if lexems[endIdx].V == ")" {
						return startIdx, endIdx // "ident IN (1,2,3)"
					}
					if lexems[endIdx].T != LexemComma && lexems[endIdx].T != LexemNumberLiteral && lexems[endIdx].T != LexemStringLiteral && lexems[endIdx].T != LexemBoolLiteral && lexems[endIdx].T != LexemQuestionMark {
						// Non-literal among IN arguments
						return 0, 0
					}
					endIdx++
				}
				// Closing parenthesis not found, give up
				return 0, 0
			} else if i < len(lexems)-1 && lexems[i+1].V == "?" {
				// It's "IN ?" (list parameter)
				return i - 1, i + 1 // "ident IN ?"
			}
		}
	}
	// No candidates found, give up
	return 0, 0
}

// Convert IN/NOT IN to a series of Go ==/!=
func convertInNotInLexemsForAstParser(lexems []*Lexem) ([]*Lexem, error) {
	for {
		startIdx, endIdx := findInNotInLexem(lexems)
		if endIdx == 0 {
			break
		}
		newLexems := make([]*Lexem, 0)
		newLexems = append(newLexems, lexems[0:startIdx]...)

		curValLexemIdx := startIdx + 3                               // First literal in the IN list of arguments
		newLexems = append(newLexems, &Lexem{LexemParenthesis, "("}) // Wrap our series of OR conditions with ()
		for curValLexemIdx < endIdx {
			if lexems[curValLexemIdx].T != LexemComma {
				if lexems[curValLexemIdx-1].V != "(" {
					newLexems = append(newLexems, &Lexem{LexemLogicalOp, "||"})
				}
				newLexems = append(newLexems, lexems[startIdx])
				switch lexems[startIdx+1].V {
				case "IN":
					newLexems = append(newLexems, &Lexem{LexemLogicalOp, "=="})
				case "NOT IN":
					newLexems = append(newLexems, &Lexem{LexemLogicalOp, "!="})
				default:
					return nil, fmt.Errorf("unexpected IN/NOT IN lexem, dev error: %s", lexems[startIdx+1].V)
				}
				newLexems = append(newLexems, lexems[curValLexemIdx])
			}
			curValLexemIdx++
		}
		newLexems = append(newLexems, &Lexem{LexemParenthesis, ")"}) // Wrap our series of OR conditions with ()
		newLexems = append(newLexems, lexems[endIdx+1:]...)
		lexems = newLexems
	}
	return lexems, nil
}

// Returns index of the AS in CAST(... AS <type>)
func findCastAsLexem(lexems []*Lexem) int {
	// Well, this is a leap of faith
	for i := range len(lexems) {
		if lexems[i].T == LexemAs {
			if i >= 3 && len(lexems) >= i+3 && isValidDataType(strings.ToLower(lexems[i+1].V)) && lexems[i+2].V == ")" {
				return i
			}
		}
	}
	// No candidates found, give up
	return -1
}

// Replace "AS" with a comma to make ast parser happy, "CAST(a AS int)" becomes "CAST(a,int)"
func convertCastForAstParser(lexems []*Lexem) []*Lexem {
	for {
		asIdx := findCastAsLexem(lexems)
		if asIdx == -1 {
			break
		}
		newLexems := make([]*Lexem, 0)
		newLexems = append(newLexems, lexems[0:asIdx]...)
		newLexems = append(newLexems, &Lexem{LexemComma, ","})
		newLexems = append(newLexems, lexems[asIdx+1:]...)
		lexems = newLexems
	}
	return lexems
}

// Replace "exp IN (1,2)" with "exp == cqlin(1,2)", and later, astutil will replace it with cqlin(exp,1,2)
func convertInNotInForAstParser(lexems []*Lexem) []*Lexem {
	for i := range len(lexems) {
		if lexems[i].T != LexemCqlOp || (lexems[i].V != "IN" && lexems[i].V != "NOT IN") || i >= len(lexems)-1 || (lexems[i+1].V != "?" && lexems[i+1].V != "(") {
			continue
		}
		newLexems := make([]*Lexem, 0)
		newLexems = append(newLexems, lexems[0:i]...)
		newLexems = append(newLexems, &Lexem{LexemLogicalOp, "=="})
		if lexems[i].V == "IN" {
			newLexems = append(newLexems, &Lexem{LexemIdent, "cqlin"})
		} else {
			newLexems = append(newLexems, &Lexem{LexemIdent, "cqlnotin"})
		}
		if lexems[i+1].V == "(" {
			// IN (1,2,3) -> cqlin(1,2,3),  IN (?,?) -> cqlin(?,?)
			newLexems = append(newLexems, lexems[i+1:]...)
		} else {
			// IN ? -> cqlin(?)
			newLexems = append(newLexems, &Lexem{LexemParenthesis, "("})
			newLexems = append(newLexems, lexems[i+1]) // ?
			newLexems = append(newLexems, &Lexem{LexemParenthesis, ")"})
			newLexems = append(newLexems, lexems[i+2:]...)
		}
		lexems = newLexems
	}
	return lexems
}

func getWhereExpressionLexems(s string) ([]*Lexem, string, error) {
	var l *Lexem
	parenthesisStackLen := 0
	lexems := make([]*Lexem, 0)
	for {
		// All possible stopwords for SELECT/UPDATE/DELETE
		stopWord := `(?i)(GROUP\s+BY|ORDER\s+BY|LIMIT|OFFSET|ALLOW\s+FILTERING|IF\s+EXISTS|IF)(\b)`
		l, s = getKeyword(s, stopWord, false)
		if l != nil {
			if parenthesisStackLen == 0 {
				break
			}
			l, s = getKeyword(s, stopWord, true)
			lexems = append(lexems, l)
			continue
		}
		// Stop comma - swallow if it's within parenthesis
		if l, s = getComma(s); l != nil {
			if parenthesisStackLen == 0 {
				break
			}
			lexems = append(lexems, l)
			continue
		}
		// Parenthesis
		if l, s = getParenthesis(s, true); l != nil {
			if l.V == "(" {
				parenthesisStackLen++
			} else {
				parenthesisStackLen--
			}
			lexems = append(lexems, l)
			continue
		}
		if l, s = getQuestionMark(s); l != nil {
			lexems = append(lexems, l)
			continue
		}
		if len(lexems) == 0 || lexems[len(lexems)-1].V == "(" || lexems[len(lexems)-1].T == LexemComma || lexems[len(lexems)-1].T == LexemArithmeticOp || lexems[len(lexems)-1].T == LexemLogicalOp {
			// No arithmetic op allowed here
			if l, s = getBoolLiteral(s); l != nil {
				lexems = append(lexems, l)
				continue
			}
			if l, s = getNull(s); l != nil {
				lexems = append(lexems, l)
				continue
			}
			if l, s = getStringLiteral(s); l != nil {
				lexems = append(lexems, l)
				continue
			}
			if l, s = getNumberLiteral(s); l != nil {
				lexems = append(lexems, l)
				continue
			}
		} else {
			// No literal allowed here
			if l, s = getArithmeticOp(s); l != nil {
				lexems = append(lexems, l)
				continue
			}
			if l, s = getLogicalOp(s); l != nil {
				lexems = append(lexems, l)
				continue
			}
		}
		if l, s = getAs(s); l != nil {
			lexems = append(lexems, l)
			continue
		}
		if l, s = getCqlOp(s); l != nil {
			lexems = append(lexems, l)
			continue
		}
		if l, s = getIdentOrPointedIdent(s); l != nil {
			lexems = append(lexems, l)
			continue
		}
		break
	}

	return lexems, s, nil
}

func getKeyValuePair(s string) (*KeyValuePair, string, error) {
	s = skipBlank(s)
	var l *Lexem
	if l, s = getKeyword(s, `}`, false); l != nil {
		return nil, s, nil
	}
	kvPair := KeyValuePair{}
	l, s = getStringLiteral(s)
	if l != nil {
		kvPair.K = l.V
		l, s = getKeyword(s, `:`, true)
		if l.V == ":" {
			l, s = getStringLiteral(s)
			if l != nil {
				kvPair.V = l
				return &kvPair, s, nil
			}
			l, s = getNumberLiteral(s)
			if l != nil {
				kvPair.V = l
				return &kvPair, s, nil
			}
			return nil, s, fmt.Errorf("cannot parse kv pair value: %s", s)
		}
		return nil, s, fmt.Errorf("cannot parse kv pair colon: %s", s)
	}
	return nil, s, fmt.Errorf("cannot parse kv pair key: %s", s)
}

func getKeyValuePairList(s string) ([]*KeyValuePair, string, error) {
	var l *Lexem
	l, s = getKeyword(s, `{`, true)
	if l == nil {
		return nil, s, fmt.Errorf("cannot parse kv pair list, { expected: %s", s)
	}
	kvPairList := make([]*KeyValuePair, 0)
	for {
		l, s = getComma(s)
		if l != nil {
			continue
		}
		l, s = getKeyword(s, `}`, true)
		if l != nil {
			break
		}
		var kvPair *KeyValuePair
		var err error
		kvPair, s, err = getKeyValuePair(s)
		if err != nil {
			return nil, s, fmt.Errorf("cannot parse kv pair: %s", s)
		}
		if kvPair != nil {
			kvPairList = append(kvPairList, kvPair)
		}
	}
	return kvPairList, s, nil
}

func getColumnDef(s string) (*CreateTableColumnDef, string, bool, error) {
	s = skipBlank(s)
	var l *Lexem
	if l, s = getKeyword(s, `\)`, false); l != nil {
		return nil, s, false, nil
	}
	def := CreateTableColumnDef{}
	l, s = getIdent(s)
	if l != nil {
		def.Name = l.V
		l, s = getKeyword(s, `(?i)(BIGINT|BLOB|BOOLEAN|COUNTER|DATE|DECIMAL|DOUBLE|DURATION|FLOAT|INET|INT|SMALLINT|TEXT|TIMEUUID|TIMESTAMP|TIME|TINYINT|UUID|VARCHAR|VARINT)(\b)`, true)
		if l != nil {
			typ, err := stringToType(l.V)
			if err != nil {
				return nil, s, false, fmt.Errorf("cannot parse column def type: %s", err.Error())
			}
			def.ColumnType = typ
			l, s = getKeyword(s, `(?i)(PRIMARY\s+KEY)(\b)`, true)
			if l != nil {
				// This field def has PRIMARY KEY tag right in it
				return &def, s, true, nil
			}

			return &def, s, false, nil
		}
		return nil, s, false, fmt.Errorf("cannot parse column def type: %s", s)
	}
	return nil, s, false, fmt.Errorf("cannot parse column def name: %s", s)
}

func getColumnDefList(s string) ([]*CreateTableColumnDef, string, string, error) {
	var l *Lexem
	primaryKeyField := ""
	defList := make([]*CreateTableColumnDef, 0)
	for {
		l, s = getComma(s)
		if l != nil {
			continue
		}
		// This marks the start of the PRIMARY KEY section, which may or may not be there
		l, s = getKeyword(s, `(?i)(PRIMARY\s+KEY)(\b)`, false)
		if l != nil {
			break
		}

		l, s = getParenthesis(s, false)
		if l != nil {
			if l.V == ")" {
				break
			} else {
				return nil, s, "", fmt.Errorf("cannot parse column def, unexpected open parenthesis: %s", s)
			}
		}

		var def *CreateTableColumnDef
		var err error
		var isPrimaryKey bool
		def, s, isPrimaryKey, err = getColumnDef(s)
		if err != nil {
			return nil, s, "", fmt.Errorf("cannot parse column def: %s,  %s", err.Error(), s)
		}
		if def == nil {
			return nil, s, "", fmt.Errorf("missing column def or missing PRIMARY KEY: %s", s)
		}
		if isPrimaryKey {
			if primaryKeyField != "" {
				return nil, s, "", fmt.Errorf("cannot have more than one field marked with PRIMARY KEY: %s", s)
			}
			primaryKeyField = def.Name
		}
		defList = append(defList, def)
	}
	return defList, s, primaryKeyField, nil
}

func getPartitionAndClusteringKeys(s string) ([]string, []string, string, error) {
	var l *Lexem
	l, s = getKeyword(s, `\(`, true)
	if l == nil {
		return nil, nil, s, fmt.Errorf("cannot parse keys, ( expected: %s", s)
	}
	partitionKeys := make([]string, 0)
	clusteringKeys := make([]string, 0)

	partitionKeysListComplete := false

	l, s = getKeyword(s, `\(`, true)
	if l != nil {
		for {
			l, s = getComma(s)
			if l != nil {
				continue
			}
			l, s = getKeyword(s, `\)`, true)
			if l != nil {
				break
			}
			l, s = getIdent(s)
			if l == nil {
				return nil, nil, s, fmt.Errorf("cannot parse partition key: %s", s)
			}
			partitionKeys = append(partitionKeys, l.V)
		}
		partitionKeysListComplete = true
	}
	for {
		l, s = getComma(s)
		if l != nil {
			continue
		}
		l, s = getKeyword(s, `\)`, true)
		if l != nil {
			break
		}
		l, s = getIdent(s)
		if l == nil {
			return nil, nil, s, fmt.Errorf("cannot parse partition/clustering key: %s", s)
		}

		if !partitionKeysListComplete {
			partitionKeys = append(partitionKeys, l.V)
			partitionKeysListComplete = true
		} else {
			clusteringKeys = append(clusteringKeys, l.V)
		}
	}
	return partitionKeys, clusteringKeys, s, nil
}

func getColumnSetExpressionLexems(s string) ([]*Lexem, string) {
	var l *Lexem
	parenthesisStackLen := 0
	lexems := make([]*Lexem, 0)
	for {
		stopWord := `(?i)(WHERE|IF)\b`
		l, s = getKeyword(s, stopWord, false)
		if l != nil {
			if parenthesisStackLen == 0 {
				break
			}
			l, s = getKeyword(s, stopWord, true)
			lexems = append(lexems, l)
			continue
		}
		// Stop comma - swallow if it's within parenthesis
		if l, s = getComma(s); l != nil {
			if parenthesisStackLen == 0 {
				break
			}
			lexems = append(lexems, l)
			continue
		}
		// Parenthesis
		if l, s = getParenthesis(s, true); l != nil {
			if l.V == "(" {
				parenthesisStackLen++
			} else {
				parenthesisStackLen--
			}
			lexems = append(lexems, l)
			continue
		}
		if l, s = getQuestionMark(s); l != nil {
			lexems = append(lexems, l)
			continue
		}
		if len(lexems) == 0 || lexems[len(lexems)-1].V == "(" || lexems[len(lexems)-1].T == LexemComma || lexems[len(lexems)-1].T == LexemArithmeticOp || lexems[len(lexems)-1].T == LexemLogicalOp {
			// No arithmetic op allowed here
			if l, s = getBoolLiteral(s); l != nil {
				lexems = append(lexems, l)
				continue
			}
			if l, s = getNull(s); l != nil {
				lexems = append(lexems, l)
				continue
			}
			if l, s = getStringLiteral(s); l != nil {
				lexems = append(lexems, l)
				continue
			}
			if l, s = getNumberLiteral(s); l != nil {
				lexems = append(lexems, l)
				continue
			}
		} else {
			// No literal allowed here
			if l, s = getArithmeticOp(s); l != nil {
				lexems = append(lexems, l)
				continue
			}
			if l, s = getLogicalOp(s); l != nil {
				lexems = append(lexems, l)
				continue
			}
		}
		if l, s = getCqlOp(s); l != nil {
			lexems = append(lexems, l)
			continue
		}
		// No pointed idents allowed here? I guess so.
		if l, s = getIdent(s); l != nil {
			lexems = append(lexems, l)
			continue
		}
		break
	}

	return lexems, s
}

func getColumnSetExpressions(s string) ([]*ColumnSetExp, string, error) {
	s = skipBlank(s)
	var l *Lexem
	columnsSetExpList := make([]*ColumnSetExp, 0)
	for {
		l, s = getComma(s)
		if l != nil {
			continue
		}
		l, s = getKeyword(s, `(?i)(WHERE|IF)\b`, false)
		if l != nil {
			break
		}
		l, s = getIdent(s)
		if l == nil {
			break
		}

		exp := ColumnSetExp{Name: l.V}
		l, s = getKeyword(s, `=`, true)
		if l == nil {
			return nil, s, errors.New("expected =")
		}
		exp.ExpLexems, s = getColumnSetExpressionLexems(s)
		columnsSetExpList = append(columnsSetExpList, &exp)
	}
	return columnsSetExpList, s, nil
}

func lexemsToStringForColumnNames(lexems []*Lexem) string {
	sb := strings.Builder{}
	for i, l := range lexems {
		// This handles SELECT expr AS synt_field_name
		if l.T == LexemAs {
			if len(lexems) == i+2 && lexems[i+1].T == LexemIdent {
				return lexems[i+1].V
			}
		}
		if l.T == LexemCqlOp || l.T == LexemAs {
			sb.WriteString(fmt.Sprintf(" %s ", l.V))
		} else {
			sb.WriteString(fmt.Sprintf("%s", l.V))
		}
	}
	return sb.String()
}

func lexemsToStringForColumnExpression(lexems []*Lexem) (string, error) {
	sb := strings.Builder{}
	for i, l := range lexems {
		// This handles SELECT expr AS synt_field_name
		if l.T == LexemAs {
			if len(lexems) == i+2 && lexems[i+1].T == LexemIdent {
				// Stop scanning
				break
			}
		}
		switch l.T {
		case LexemComma, LexemArithmeticOp, LexemLogicalOp, LexemBoolLiteral, LexemNumberLiteral, LexemParenthesis:
			sb.WriteString(fmt.Sprintf("%s", l.V))
		case LexemAsterisk, LexemPointedAsterisk:
			if i >= 2 && (lexems[i-2].V == "count" || lexems[i-2].V == "COUNT") && lexems[i-1].V == "(" && i < len(lexems)-1 && lexems[i+1].V == ")" {
				// Write nothing, let it be just count()
			} else if i == 0 && len(lexems) == 1 {
				// This is just a SELECT * FROM ...
				sb.WriteString(fmt.Sprintf("%s", strings.ReplaceAll(l.V, "*", "ALL_FIELDS")))
			} else {
				return "", fmt.Errorf("unexpected asterisk lexem (%d,%s), not expected here", l.T, l.V)
			}
		case LexemIdent, LexemPointedIdent:
			if isValidDataType(l.V) {
				// CQL data types are UPPERCASE
				sb.WriteString(fmt.Sprintf("%s", strings.ToUpper(l.V)))
			} else if i < len(lexems)-1 && lexems[i+1].V == "(" {
				// functions are lowercase
				sb.WriteString(fmt.Sprintf("%s", strings.ToLower(l.V)))
			} else {
				sb.WriteString(fmt.Sprintf("%s", l.V))
			}
		case LexemStringLiteral:
			sb.WriteString(fmt.Sprintf("`%s`", l.V))
		// case LexemAs: Not used?
		// 	sb.WriteString(",")
		case LexemNull:
			sb.WriteString("NULL") // GocqlmemEvalConstants will take care of this
		case LexemSemicolon, LexemCqlOp, LexemKeyword:
			return "", fmt.Errorf("unexpected lexem (%d,%s)", l.T, l.V)
		default:
			return "", fmt.Errorf("unknown lexem (%d,%s)", l.T, l.V)
		}
	}
	return sb.String(), nil
}

func lexemsToAstExpr(lexems []*Lexem) (ast.Expr, error) {
	s, err := lexemsToStringForColumnExpression(lexems)
	if err != nil {
		return nil, err
	}
	exp, err := parser.ParseExpr(s)
	return exp, err
}

func parseCreateKeyspace(s string) (*CommandCreateKeyspace, string, error) {
	var l *Lexem
	l, s = getKeyword(s, `(?i)CREATE\s+KEYSPACE\b`, true)
	if l == nil {
		return nil, s, fmt.Errorf("expected CREATE KEYSPACE: %s", s)
	}
	cmd := CommandCreateKeyspace{}
	l, s = getKeyword(s, `(?i)IF\s+NOT\s+EXISTS\b`, true)
	if l != nil {
		cmd.IfNotExists = true
	}
	l, s = getIdent(s)
	if l != nil {
		cmd.KeyspaceName = l.V
	}
	l, s = getKeyword(s, `(?i)WITH\s+REPLICATION\b`, true)
	if l != nil {
		var err error
		cmd.WithReplication, s, err = getKeyValuePairList(s)
		if err != nil {
			return nil, s, fmt.Errorf("cannot parse with replication: %s", err.Error())
		}
	}

	return &cmd, s, nil
}

func parseUseKeyspace(s string) (*CommandUseKeyspace, string, error) {
	var l *Lexem
	l, s = getKeyword(s, `(?i)USE\b`, true)
	if l == nil {
		return nil, s, fmt.Errorf("expected USE: %s", s)
	}
	cmd := CommandUseKeyspace{}
	l, s = getIdent(s)
	if l != nil {
		cmd.KeyspaceName = l.V
	}
	return &cmd, s, nil
}

func parseDropKeyspace(s string) (*CommandDropKeyspace, string, error) {
	var l *Lexem
	l, s = getKeyword(s, `(?i)DROP\s+KEYSPACE\b`, true)
	if l == nil {
		return nil, s, fmt.Errorf("expected DROP KEYSPACE: %s", s)
	}
	cmd := CommandDropKeyspace{}
	l, s = getKeyword(s, `(?i)IF\s+EXISTS\b`, true)
	if l != nil {
		cmd.IfExists = true
	}
	l, s = getIdent(s)
	if l != nil {
		cmd.KeyspaceName = l.V
	}
	return &cmd, s, nil
}

func parseCreateTable(s string) (*CommandCreateTable, string, error) {
	var l *Lexem
	var err error
	var primaryKeyField string
	l, s = getKeyword(s, `(?i)CREATE\s+TABLE\b`, true)
	if l == nil {
		return nil, s, fmt.Errorf("expected CREATE TABLE: %s", s)
	}
	cmd := CommandCreateTable{}
	l, s = getKeyword(s, `(?i)IF\s+NOT\s+EXISTS\b`, true)
	if l != nil {
		cmd.IfNotExists = true
	}
	l, s = getIdentOrPointedIdent(s)
	if l == nil {
		return nil, s, fmt.Errorf("expected table name ident: %s", s)
	}
	if l.T == LexemPointedIdent {
		ksTable := strings.Split(l.V, ".")
		cmd.CtxKeyspace = ksTable[0]
		cmd.TableName = ksTable[1]
	} else {
		cmd.TableName = l.V
	}

	l, s = getKeyword(s, `\(`, true)
	if l == nil {
		return nil, s, fmt.Errorf("cannot parse column def list, ( expected: %s", s)
	}

	cmd.ColumnDefs, s, primaryKeyField, err = getColumnDefList(s)
	if err != nil {
		return nil, s, err
	}

	if primaryKeyField == "" {

		l, s = getKeyword(s, `(?i)PRIMARY\s+KEY\b`, true)
		if l == nil {
			return nil, s, fmt.Errorf("expected PRIMARY KEY: %s", s)
		}

		cmd.PartitionKeyColumns, cmd.ClusteringKeyColumns, s, err = getPartitionAndClusteringKeys(s)
		if err != nil {
			return nil, s, err
		}
	} else {
		cmd.PartitionKeyColumns = []string{primaryKeyField}
	}
	l, s = getKeyword(s, `\)`, true)
	if l == nil {
		return nil, s, fmt.Errorf("cannot parse column def list, ) expected: %s", s)
	}

	l, s = getKeyword(s, `(?i)WITH\s+CLUSTERING\s+ORDER\s+BY\b`, true)
	if l != nil {
		l, s = getKeyword(s, `\(`, true)
		if l == nil {
			return nil, s, fmt.Errorf("cannot parse WITH CLUSTERING ORDER BY, ( expected: %s", s)
		}
		cmd.ClusteringOrderBy = make([]*OrderByField, 0)
		for {
			l, s = getComma(s)
			if l != nil {
				continue
			}
			l, s = getKeyword(s, `\)`, true)
			if l != nil {
				break
			}
			var lField *Lexem
			lField, s = getIdent(s)
			if lField == nil {
				return nil, s, fmt.Errorf("expected clustering order by ident: %s", s)
			}
			var lAscDesc *Lexem
			lAscDesc, s = getKeyword(s, `(?i)(ASC|DESC)(\b)`, true)
			if lAscDesc == nil {
				return nil, s, fmt.Errorf("expected clustering order by asc or desc: %s", s)
			}

			clusteringOrder := stringToClusteringOrderType(lAscDesc.V)
			if clusteringOrder == ClusteringOrderNone {
				return nil, s, fmt.Errorf("expected clustering order by asc or desc, got %s", lAscDesc.V)
			}
			cmd.ClusteringOrderBy = append(cmd.ClusteringOrderBy, &OrderByField{lField.V, clusteringOrder})
		}
		for _, clustOrderBy := range cmd.ClusteringOrderBy {
			var clusteringKeyFieldFound bool
			for _, clusteringKeyColumn := range cmd.ClusteringKeyColumns {
				if clusteringKeyColumn == clustOrderBy.FieldName {
					clusteringKeyFieldFound = true
					break
				}
			}
			if !clusteringKeyFieldFound {
				return nil, s, fmt.Errorf("clustering order field %s specified, but it's not among clustering keys", clustOrderBy.FieldName)
			}
		}
	}

	if len(cmd.ColumnDefs) == 0 {
		return nil, s, errors.New("cannot parse CREATE TABLE with empty columnn def list")
	}

	if len(cmd.PartitionKeyColumns) == 0 {
		return nil, s, errors.New("cannot parse CREATE TABLE with empty partition column list")
	}

	for _, fieldName := range cmd.PartitionKeyColumns {
		var fieldFound bool
		for i := range len(cmd.ColumnDefs) {
			if fieldName == cmd.ColumnDefs[i].Name {
				fieldFound = true
				break
			}
		}
		if !fieldFound {
			return nil, s, fmt.Errorf("partition key %s not found in column definitions", fieldName)
		}
	}

	for _, fieldName := range cmd.ClusteringKeyColumns {
		var fieldFound bool
		for i := range len(cmd.ColumnDefs) {
			if fieldName == cmd.ColumnDefs[i].Name {
				fieldFound = true
				break
			}
		}
		if !fieldFound {
			return nil, s, fmt.Errorf("clustering key %s not found in column definitions", fieldName)
		}
	}

	fieldMap := map[string]struct{}{}
	for _, fieldName := range cmd.PartitionKeyColumns {
		if _, ok := fieldMap[fieldName]; ok {
			return nil, s, fmt.Errorf("partition key %s duplication", fieldName)
		}
		fieldMap[fieldName] = struct{}{}
	}
	for _, fieldName := range cmd.ClusteringKeyColumns {
		if _, ok := fieldMap[fieldName]; ok {
			return nil, s, fmt.Errorf("clustering key %s duplication", fieldName)
		}
		fieldMap[fieldName] = struct{}{}
	}

	return &cmd, s, nil
}

func parseTruncateTable(s string) (*CommandTruncateTable, string, error) {
	var l *Lexem
	l, s = getKeyword(s, `(?i)TRUNCATE\b`, true)
	if l == nil {
		return nil, s, fmt.Errorf("expected TRUNCATE: %s", s)
	}
	cmd := CommandTruncateTable{}
	l, s = getIdentOrPointedIdent(s)
	if l == nil {
		return nil, s, fmt.Errorf("expected table name ident: %s", s)
	}
	if l.T == LexemPointedIdent {
		ksTable := strings.Split(l.V, ".")
		cmd.CtxKeyspace = ksTable[0]
		cmd.TableName = ksTable[1]
	} else {
		cmd.TableName = l.V
	}

	return &cmd, s, nil
}

func parseDropTable(s string) (*CommandDropTable, string, error) {
	var l *Lexem
	l, s = getKeyword(s, `(?i)DROP\s+TABLE\b`, true)
	if l == nil {
		return nil, s, fmt.Errorf("expected DROP TABLE: %s", s)
	}
	cmd := CommandDropTable{}
	l, s = getKeyword(s, `(?i)IF\s+EXISTS\b`, true)
	if l != nil {
		cmd.IfExists = true
	}
	l, s = getIdentOrPointedIdent(s)
	if l == nil {
		return nil, s, fmt.Errorf("expected table name ident: %s", s)
	}
	if l.T == LexemPointedIdent {
		ksTable := strings.Split(l.V, ".")
		cmd.CtxKeyspace = ksTable[0]
		cmd.TableName = ksTable[1]
	} else {
		cmd.TableName = l.V
	}

	return &cmd, s, nil
}

func addPreparedQueryParamsToMap(valMap eval.VarValuesMap, preparedQueryParams []any) error {
	if len(preparedQueryParams) > 0 {
		valMap["params"] = map[string]any{}
		for paramIdx := range preparedQueryParams {
			paramSlice := reflect.ValueOf(preparedQueryParams[paramIdx])
			if paramSlice.Kind() == reflect.Slice {
				for i := range paramSlice.Len() {
					internalTypedVal, err := castToInternalType(paramSlice.Index(i).Interface())
					if err != nil {
						return err
					}
					valMap["params"][fmt.Sprintf("param%03d_%03d", paramIdx, i)] = internalTypedVal
				}
			} else {
				internalTypedVal, err := castToInternalType(preparedQueryParams[paramIdx])
				if err != nil {
					return err
				}
				valMap["params"][fmt.Sprintf("param%03d", paramIdx)] = internalTypedVal
			}
		}
	}
	return nil
}

func replaceQuestionMarksWithParamNamesInLexems(paramIdx int, lexems []*Lexem, preparedQueryParams []any) (int, []*Lexem, error) {
	for lexemIdx := 0; lexemIdx < len(lexems); lexemIdx++ {
		if lexems[lexemIdx].T != LexemQuestionMark {
			continue
		}
		if paramIdx >= len(preparedQueryParams) {
			return -1, nil, fmt.Errorf("not enough prepared params supplied: %d", len(preparedQueryParams))
		}
		paramSlice := reflect.ValueOf(preparedQueryParams[paramIdx])
		if paramSlice.Kind() == reflect.Slice {
			insertedLexems := []*Lexem{}
			for i := range paramSlice.Len() {
				if i != 0 {
					insertedLexems = append(insertedLexems, &Lexem{LexemComma, ","})
				}
				insertedLexems = append(insertedLexems, &Lexem{LexemPointedIdent, fmt.Sprintf("params.param%03d_%03d", paramIdx, i)})
			}
			newLexems := make([]*Lexem, 0)
			newLexems = append(newLexems, lexems[:lexemIdx]...)
			newLexems = append(newLexems, insertedLexems...)
			newLexemIdx := len(newLexems) // Reset the counter because we have just replcaed the slice we are working with
			newLexems = append(newLexems, lexems[lexemIdx+1:]...)
			lexems = newLexems
			lexemIdx = newLexemIdx

		} else {
			lexems[lexemIdx] = &Lexem{LexemPointedIdent, fmt.Sprintf("params.param%03d", paramIdx)}
		}
		paramIdx++
	}
	return paramIdx, lexems, nil
}

func convertIns(exp ast.Expr) (ast.Expr, error) {
	modifiedNode := astutil.Apply(exp, func(cursor *astutil.Cursor) bool {
		eqExp, ok := cursor.Node().(*ast.BinaryExpr)
		if !ok || eqExp.Op != token.EQL {
			return true
		}
		callExp, ok := eqExp.Y.(*ast.CallExpr)
		if !ok {
			return true
		}
		funExp := callExp.Fun
		funIdentExp, ok := funExp.(*ast.Ident)
		if !ok {
			return true
		}
		if funIdentExp.Name != "cqlin" && funIdentExp.Name != "cqlnotin" {
			return true
		}

		// Build a new node instead of the "==" BinaryExpr, make it from the call exp we have on hand
		callExp.Args = slices.Insert(callExp.Args, 0, eqExp.X)
		cursor.Replace(callExp)
		return true
	}, nil)

	modifiedExpr, ok := modifiedNode.(ast.Expr)
	if !ok {
		return nil, fmt.Errorf("cannot cast modified ast.Node to ast.Expr")
	}
	return modifiedExpr, nil
}

func parseSelect(s string, preparedQueryParams []any) (*CommandSelect, string, error) {
	var l *Lexem
	var err error
	l, s = getKeyword(s, `(?i)SELECT\b`, true)
	if l == nil {
		return nil, s, fmt.Errorf("expected SELECT: %s", s)
	}
	cmd := CommandSelect{}
	l, s = getKeyword(s, `(?i)DISTINCT\b`, true)
	if l != nil {
		cmd.Distinct = true
	}
	l, s = getKeyword(s, `(?i)JSON\b`, true)
	if l != nil {
		return nil, s, errors.New("JSON not supported")
	}
	cmd.SelectExpLexems, s = getSelectExpressions(s)
	if len(cmd.SelectExpLexems) == 0 {
		return nil, s, fmt.Errorf("expected select expressions: %s", s)
	}
	l, s = getKeyword(s, `(?i)FROM\b`, true)
	if l == nil {
		return nil, s, fmt.Errorf("expected FROM: %s", s)
	}
	l, s = getIdentOrPointedIdent(s)
	if l == nil {
		return nil, s, fmt.Errorf("expected from table ident: %s", s)
	}
	if l.T == LexemPointedIdent {
		ksTable := strings.Split(l.V, ".")
		cmd.CtxKeyspace = ksTable[0]
		cmd.TableName = ksTable[1]
	} else {
		cmd.TableName = l.V
	}

	l, s = getKeyword(s, `(?i)WHERE\b`, true)
	if l != nil {
		cmd.WhereExpLexems, s, err = getWhereExpressionLexems(s)
		if err != nil {
			return nil, s, fmt.Errorf("cannot parse where expression: %s", err.Error())
		}
		if cmd.WhereExpLexems == nil {
			return nil, s, fmt.Errorf("expected where expression: %s", s)
		}
	}
	l, s = getKeyword(s, `(?i)GROUP\s+BY\b`, true)
	if l != nil {
		return nil, s, errors.New("GROUP BY not supported")
	}
	l, s = getKeyword(s, `(?i)ORDER\s+BY\b`, true)
	if l != nil {
		cmd.OrderByFields = make([]*OrderByField, 0)
		for {
			l, s = getComma(s)
			if l != nil {
				continue
			}
			l, s = getKeyword(s, `(?i)(PER\s+PARTITION\s+LIMIT|LIMIT|ALLOW\sFILTERING|OFFSET)(\b)`, false)
			if l != nil {
				break
			}
			var lField *Lexem
			lField, s = getIdent(s)
			if lField == nil {
				return nil, s, fmt.Errorf("expected order by ident: %s", s)
			}
			var lAscDesc *Lexem
			lAscDesc, s = getKeyword(s, `(?i)(ASC|DESC)(\b)`, true)
			if lAscDesc == nil {
				return nil, s, fmt.Errorf("expected order by asc or desc: %s", s)
			}
			clusteringOrder := stringToClusteringOrderType(lAscDesc.V)
			if clusteringOrder == ClusteringOrderNone {
				return nil, s, fmt.Errorf("expected order by asc or desc, got %s", lAscDesc.V)
			}
			cmd.OrderByFields = append(cmd.OrderByFields, &OrderByField{lField.V, clusteringOrder})
		}
	}
	l, s = getKeyword(s, `(?i)PER\s+PARTITION\s+LIMIT\b`, true)
	if l != nil {
		return nil, s, errors.New("PER PARTITION LIMIT not supported")
	}
	l, s = getKeyword(s, `(?i)LIMIT\b`, true)
	if l != nil {
		cmd.Limit, s = getNumberLiteral(s)
		if cmd.Limit == nil {
			return nil, s, fmt.Errorf("expected limit number: %s", s)
		}
	}
	l, s = getKeyword(s, `(?i)OFFSET\b`, true)
	if l != nil {
		return nil, s, errors.New("OFFSET not supported")
	}
	l, s = getKeyword(s, `(?i)ALLOW\sFILTERING\b`, true)
	if l != nil {
		return nil, s, errors.New("ALLOW FILTERING not supported")
	}

	// Initialize column names before any tricks/replacements
	cmd.SelectExpNames = []string{}
	for _, selectExp := range cmd.SelectExpLexems {
		cmd.SelectExpNames = append(cmd.SelectExpNames, lexemsToStringForColumnNames(selectExp))
	}

	paramIdx := 0
	// CAST/IN/preparedQueryParams in select column expression:
	for selectExpIdx := range len(cmd.SelectExpLexems) {
		// Convert AS and IN/NOT IN to functions so Go parser can work with them
		cmd.SelectExpLexems[selectExpIdx] = convertCastForAstParser(cmd.SelectExpLexems[selectExpIdx])
		cmd.SelectExpLexems[selectExpIdx] = convertInNotInForAstParser(cmd.SelectExpLexems[selectExpIdx])
		// Replace all question marks with prepared param names: in select expressions
		if paramIdx, cmd.SelectExpLexems[selectExpIdx], err = replaceQuestionMarksWithParamNamesInLexems(paramIdx, cmd.SelectExpLexems[selectExpIdx], preparedQueryParams); err != nil {
			return nil, s, err
		}
	}

	// CAST/IN/preparedQueryParams in where expression:
	// Convert AS and IN/NOT IN to functions so Go parser can work with them
	cmd.WhereExpLexems = convertCastForAstParser(cmd.WhereExpLexems)
	cmd.WhereExpLexems = convertInNotInForAstParser(cmd.WhereExpLexems)
	// Replace all question marks with prepared param names: in select expressions
	if paramIdx, cmd.WhereExpLexems, err = replaceQuestionMarksWithParamNamesInLexems(paramIdx, cmd.WhereExpLexems, preparedQueryParams); err != nil {
		return nil, s, err
	}

	// Lexems to expressions: select column expressions
	cmd.SelectExpAsts = []ast.Expr{}
	for selectExpIdx, selectExp := range cmd.SelectExpLexems {
		astExp, err := lexemsToAstExpr(selectExp)
		if err != nil {
			return nil, s, fmt.Errorf("cannot build ast from select expression: %s", err.Error())
		}
		if astExp, err = convertIns(astExp); err != nil {
			return nil, s, fmt.Errorf("cannot convert ins for select expression %d: %s", selectExpIdx, err.Error())
		}
		cmd.SelectExpAsts = append(cmd.SelectExpAsts, astExp)
	}

	// Lexems to expressions: where expression
	if len(cmd.WhereExpLexems) > 0 {
		cmd.WhereExpAst, err = lexemsToAstExpr(cmd.WhereExpLexems)
		if err != nil {
			return nil, s, fmt.Errorf("cannot build ast from select where expression: %s", err.Error())
		}
		if cmd.WhereExpAst, err = convertIns(cmd.WhereExpAst); err != nil {
			return nil, s, fmt.Errorf("cannot convert ins for select where expression: %s", err.Error())
		}
	}

	return &cmd, s, nil
}

func getInsertExpression(s string) ([]*Lexem, string) {
	var l *Lexem
	exp := make([]*Lexem, 0)
	for {
		l, s = getKeyword(s, `(?i)IF\s(NOT\sEXISTS|USING|TIMESTAMP)(\b)`, false)
		if l != nil {
			break
		}
		if l, s = getQuestionMark(s); l != nil {
			exp = append(exp, l)
			continue
		}
		if len(exp) == 0 || exp[len(exp)-1].V == "(" || exp[len(exp)-1].T == LexemComma || exp[len(exp)-1].T == LexemArithmeticOp || exp[len(exp)-1].T == LexemLogicalOp {
			// No arithmetic op allowed here
			if l, s = getBoolLiteral(s); l != nil {
				exp = append(exp, l)
				continue
			}
			if l, s = getNull(s); l != nil {
				exp = append(exp, l)
				continue
			}
			if l, s = getStringLiteral(s); l != nil {
				exp = append(exp, l)
				continue
			}
			if l, s = getNumberLiteral(s); l != nil {
				exp = append(exp, l)
				continue
			}
		} else {
			// No literal allowed here
			if l, s = getArithmeticOp(s); l != nil {
				exp = append(exp, l)
				continue
			}
			if l, s = getLogicalOp(s); l != nil {
				exp = append(exp, l)
				continue
			}
		}
		if l, s = getAs(s); l != nil {
			exp = append(exp, l)
			continue
		}
		if l, s = getParenthesis(s, true); l != nil {
			exp = append(exp, l)
			continue
		}
		if l, s = getAsteriskOrPointedAsterisk(s); l != nil {
			exp = append(exp, l)
			continue
		}
		if l, s = getIdentOrPointedIdent(s); l != nil {
			exp = append(exp, l)
			continue
		}
		break
	}
	return exp, s
}

func getInsertExpressions(s string) ([][]*Lexem, string) {
	var l *Lexem
	exps := make([][]*Lexem, 0)
	for {
		l, s = getComma(s)
		if l != nil {
			continue
		}
		var exp []*Lexem
		exp, s = getInsertExpression(s)
		if len(exp) == 0 {
			break
		}
		exp = convertCastForAstParser(exp)
		exps = append(exps, exp)
	}

	// Remove the very last ")", we have just harvetesd it as part of the expression
	if len(exps) > 0 {
		lastExp := exps[len(exps)-1]
		if len(lastExp) > 0 && lastExp[len(lastExp)-1].V == ")" {
			exps[len(exps)-1] = exps[len(exps)-1][:len(lastExp)-1]
		}
		// If this makes last column value empty, that mean there were no value at all
		if len(exps[len(exps)-1]) == 0 {
			exps = exps[:len(exps)-1]
		}
	}

	return exps, s
}

func parseInsert(s string, preparedQueryParams []any) (*CommandInsert, string, error) {
	var l *Lexem
	l, s = getKeyword(s, `(?i)INSERT\s+INTO\b`, true)
	if l == nil {
		return nil, s, fmt.Errorf("expected INSERT INTO: %s", s)
	}
	cmd := CommandInsert{}
	l, s = getIdentOrPointedIdent(s)
	if l == nil {
		return nil, s, fmt.Errorf("expected update table ident: %s", s)
	}
	if l.T == LexemPointedIdent {
		ksTable := strings.Split(l.V, ".")
		cmd.CtxKeyspace = ksTable[0]
		cmd.TableName = ksTable[1]
	} else {
		cmd.TableName = l.V
	}

	l, s = getKeyword(s, `\(`, true)
	if l == nil {
		return nil, s, fmt.Errorf("expected (: %s", s)
	}
	cmd.ColumnNames = make([]string, 0)
	for {
		l, s = getComma(s)
		if l != nil {
			continue
		}
		l, s = getKeyword(s, `\)`, false)
		if l != nil {
			break
		}
		l, s = getIdent(s)
		if l == nil {
			return nil, s, fmt.Errorf("expected column name: %s", s)
		}
		cmd.ColumnNames = append(cmd.ColumnNames, l.V)
	}
	l, s = getKeyword(s, `\)`, true)
	if l == nil {
		return nil, s, fmt.Errorf("expected ): %s", s)
	}

	l, s = getKeyword(s, `(?i)VALUES\b`, true)
	if l == nil {
		return nil, s, fmt.Errorf("expected VALUES: %s", s)
	}

	l, s = getKeyword(s, `\(`, true)
	if l == nil {
		return nil, s, fmt.Errorf("expected (: %s", s)
	}

	cmd.ColumnValueLexems, s = getInsertExpressions(s)

	l, s = getKeyword(s, `(?i)IF\s+NOT\s+EXISTS\b`, true)
	if l != nil {
		cmd.IfNotExists = true
	}

	if len(cmd.ColumnNames) == 0 {
		return nil, s, errors.New("column list cannot be empty")
	}

	if len(cmd.ColumnNames) != len(cmd.ColumnValueLexems) {
		return nil, s, fmt.Errorf("value list length (%d) should match column list length (%d)", len(cmd.ColumnValueLexems), len(cmd.ColumnNames))
	}

	// Replace all question marks with prepared params
	paramIdx := 0
	paramValues := eval.VarValuesMap{}
	paramValues["params"] = map[string]any{}
	for columnValueIdx := range len(cmd.ColumnValueLexems) {
		for lexemIdx := range cmd.ColumnValueLexems[columnValueIdx] {
			if cmd.ColumnValueLexems[columnValueIdx][lexemIdx].T == LexemQuestionMark {
				if paramIdx >= len(preparedQueryParams) {
					return nil, s, fmt.Errorf("not enough prepared params supplied: %d", len(preparedQueryParams))
				}
				paramName := fmt.Sprintf("param%03d", paramIdx)
				cmd.ColumnValueLexems[columnValueIdx][lexemIdx] = &Lexem{LexemPointedIdent, "params." + paramName}
				paramValues["params"][paramName] = preparedQueryParams[paramIdx]
				paramIdx++
			}
		}
	}

	// We do not need any table data to calculate inserted values, so do it here right away
	cmd.ColumnValues = make([]any, len(cmd.ColumnValueLexems))
	for i := range len(cmd.ColumnValueLexems) {
		var err error
		colValueAst, err := lexemsToAstExpr(cmd.ColumnValueLexems[i])
		if err != nil {
			return nil, s, fmt.Errorf("cannot build ast from insert value expression, column %d: %s", i, err.Error())
		}
		eCtx := eval.NewPlainEvalCtx(GocqlmemEvalFunctions, GocqlmemEvalConstants, paramValues)
		cmd.ColumnValues[i], err = eCtx.Eval(colValueAst)
		if err != nil {
			return nil, s, fmt.Errorf("cannot calculate column value from insert value expression, column %d: %s", i, err.Error())
		}
	}
	return &cmd, s, nil
}

func parseUpdate(s string, preparedQueryParams []any) (*CommandUpdate, string, error) {
	var l *Lexem
	l, s = getKeyword(s, `(?i)UPDATE\b`, true)
	if l == nil {
		return nil, s, fmt.Errorf("expected UPDATE: %s", s)
	}
	cmd := CommandUpdate{}
	l, s = getIdentOrPointedIdent(s)
	if l == nil {
		return nil, s, fmt.Errorf("expected update table ident: %s", s)
	}
	if l.T == LexemPointedIdent {
		ksTable := strings.Split(l.V, ".")
		cmd.CtxKeyspace = ksTable[0]
		cmd.TableName = ksTable[1]
	} else {
		cmd.TableName = l.V
	}

	l, s = getKeyword(s, `(?i)SET\b`, true)
	if l == nil {
		return nil, s, fmt.Errorf("expected SET: %s", s)
	}

	var err error
	cmd.ColumnSetExpressions, s, err = getColumnSetExpressions(s)
	if err != nil {
		return nil, s, err
	}

	l, s = getKeyword(s, `(?i)WHERE\b`, true)
	if l != nil {
		cmd.WhereExpLexems, s, err = getWhereExpressionLexems(s)
		if err != nil {
			return nil, s, fmt.Errorf("cannot parse where expression: %s", err.Error())
		}
		if cmd.WhereExpLexems == nil {
			return nil, s, fmt.Errorf("expected where expression: %s", s)
		}
	}

	l, s = getKeyword(s, `(?i)IF\s+EXISTS\b`, true)
	if l != nil {
		cmd.IfExists = true
	}

	l, s = getKeyword(s, `(?i)IF\b`, true)
	if l != nil {
		cmd.IfExpLexems, s, err = getWhereExpressionLexems(s)
		if err != nil {
			return nil, s, fmt.Errorf("cannot parse if expression: %s", err.Error())
		}
		if cmd.IfExpLexems == nil {
			return nil, s, fmt.Errorf("expected if expression: %s", s)
		}
	}

	// Replace all question marks with prepared params: in column expressions
	paramIdx := 0
	// CAST/IN/preparedQueryParams in select column expression:
	for selectExpIdx := range len(cmd.ColumnSetExpressions) {
		// Convert AS and IN/NOT IN to functions so Go parser can work with them
		cmd.ColumnSetExpressions[selectExpIdx].ExpLexems = convertCastForAstParser(cmd.ColumnSetExpressions[selectExpIdx].ExpLexems)
		cmd.ColumnSetExpressions[selectExpIdx].ExpLexems = convertInNotInForAstParser(cmd.ColumnSetExpressions[selectExpIdx].ExpLexems)
		// Replace all question marks with prepared param names: in select expressions
		if paramIdx, cmd.ColumnSetExpressions[selectExpIdx].ExpLexems, err = replaceQuestionMarksWithParamNamesInLexems(paramIdx, cmd.ColumnSetExpressions[selectExpIdx].ExpLexems, preparedQueryParams); err != nil {
			return nil, s, err
		}
	}

	// for columnSetExpIdx := range len(cmd.ColumnSetExpressions) {
	// 	for lexemIdx := range cmd.ColumnSetExpressions[columnSetExpIdx].ExpLexems {
	// 		if cmd.ColumnSetExpressions[columnSetExpIdx].ExpLexems[lexemIdx].T == LexemQuestionMark {
	// 			if paramIdx >= len(preparedQueryParams) {
	// 				return nil, s, fmt.Errorf("not enough prepared params supplied: %d", len(preparedQueryParams))
	// 			}
	// 			cmd.ColumnSetExpressions[columnSetExpIdx].ExpLexems[lexemIdx] = &Lexem{LexemPointedIdent, fmt.Sprintf("params.param%03d", paramIdx)}
	// 			paramIdx++
	// 		}
	// 	}
	// }

	// CAST/IN/preparedQueryParams in where expression:
	// Convert AS and IN/NOT IN to functions so Go parser can work with them
	cmd.WhereExpLexems = convertCastForAstParser(cmd.WhereExpLexems)
	cmd.WhereExpLexems = convertInNotInForAstParser(cmd.WhereExpLexems)
	// Replace all question marks with prepared param names: in where expression
	if paramIdx, cmd.WhereExpLexems, err = replaceQuestionMarksWithParamNamesInLexems(paramIdx, cmd.WhereExpLexems, preparedQueryParams); err != nil {
		return nil, s, err
	}

	cmd.IfExpLexems = convertCastForAstParser(cmd.IfExpLexems)
	cmd.IfExpLexems = convertInNotInForAstParser(cmd.IfExpLexems)
	// Replace all question marks with prepared param names: in if expression
	if paramIdx, cmd.IfExpLexems, err = replaceQuestionMarksWithParamNamesInLexems(paramIdx, cmd.IfExpLexems, preparedQueryParams); err != nil {
		return nil, s, err
	}

	// And in where expression
	// if len(cmd.WhereExpLexems) > 0 {
	// 	for lexemIdx := range cmd.WhereExpLexems {
	// 		if cmd.WhereExpLexems[lexemIdx].T == LexemQuestionMark {
	// 			if paramIdx >= len(preparedQueryParams) {
	// 				return nil, s, fmt.Errorf("not enough prepared params supplied: %d", len(preparedQueryParams))
	// 			}
	// 			cmd.WhereExpLexems[lexemIdx] = &Lexem{LexemPointedIdent, fmt.Sprintf("params.param%03d", paramIdx)}
	// 			paramIdx++
	// 		}
	// 	}
	// }

	// Lexems to expressions: set expressions
	cmd.ColumnSetExpAsts = make([]ast.Expr, len(cmd.ColumnSetExpressions))
	for setExpIdx, columnSetExp := range cmd.ColumnSetExpressions {
		astExp, err := lexemsToAstExpr(columnSetExp.ExpLexems)
		if err != nil {
			return nil, s, fmt.Errorf("cannot build ast from column set expression: %s", err.Error())
		}
		if astExp, err = convertIns(astExp); err != nil {
			return nil, s, fmt.Errorf("cannot convert ins for select expression %d: %s", setExpIdx, err.Error())
		}
		cmd.ColumnSetExpAsts[setExpIdx] = astExp
	}

	// Lexems to expressions: where expression
	if len(cmd.WhereExpLexems) > 0 {
		cmd.WhereExpAst, err = lexemsToAstExpr(cmd.WhereExpLexems)
		if err != nil {
			return nil, s, fmt.Errorf("cannot build ast from update where expression: %s", err.Error())
		}
		if cmd.WhereExpAst, err = convertIns(cmd.WhereExpAst); err != nil {
			return nil, s, fmt.Errorf("cannot convert ins for update where expression: %s", err.Error())
		}
	}

	if len(cmd.IfExpLexems) > 0 {
		cmd.IfExpAst, err = lexemsToAstExpr(cmd.IfExpLexems)
		if err != nil {
			return nil, s, fmt.Errorf("cannot build ast from if expression: %s", err.Error())
		}
		if cmd.IfExpAst, err = convertIns(cmd.IfExpAst); err != nil {
			return nil, s, fmt.Errorf("cannot convert ins for update if expression: %s", err.Error())
		}
	}

	return &cmd, s, nil
}

func parseDelete(s string, preparedQueryParams []interface{}) (*CommandDelete, string, error) {
	var l *Lexem
	var err error
	cmd := CommandDelete{ColumnsToDelete: []string{}}
	l, s = getKeyword(s, `(?i)DELETE\b`, true)
	if l == nil {
		return nil, s, fmt.Errorf("expected DELETE: %s", s)
	}
	for {
		l, s = getKeyword(s, `(?i)FROM\b`, false)
		if l != nil {
			break
		}
		l, s = getComma(s)
		if l != nil {
			continue
		}
		l, s = getIdentOrPointedIdent(s)
		if l != nil {
			cmd.ColumnsToDelete = append(cmd.ColumnsToDelete, l.V)
			continue
		}
		return nil, s, fmt.Errorf("expected FROM or column name: %s", s)
	}
	l, s = getKeyword(s, `(?i)FROM\b`, true)
	if l == nil {
		return nil, s, fmt.Errorf("expected FROM: %s", s)
	}
	l, s = getIdentOrPointedIdent(s)
	if l == nil {
		return nil, s, fmt.Errorf("expected update table ident: %s", s)
	}
	if l.T == LexemPointedIdent {
		ksTable := strings.Split(l.V, ".")
		cmd.CtxKeyspace = ksTable[0]
		cmd.TableName = ksTable[1]
	} else {
		cmd.TableName = l.V
	}

	l, s = getKeyword(s, `(?i)WHERE\b`, true)
	if l != nil {
		cmd.WhereExpLexems, s, err = getWhereExpressionLexems(s)
		if err != nil {
			return nil, s, fmt.Errorf("cannot parse where expression: %s", err.Error())
		}
		if cmd.WhereExpLexems == nil {
			return nil, s, fmt.Errorf("expected where expression: %s", s)
		}
	}

	l, s = getKeyword(s, `(?i)IF\s+EXISTS\b`, true)
	if l != nil {
		cmd.IfExists = true
	}

	l, s = getKeyword(s, `(?i)IF\b`, true)
	if l != nil {
		return nil, s, fmt.Errorf("IF not upported, it affects performance: %s", s)
	}

	// Verify table name in supplied column names, strip table name if needed
	for i := range len(cmd.ColumnsToDelete) {
		parts := strings.Split(cmd.ColumnsToDelete[i], ".")
		if len(parts) == 2 {
			if parts[0] != cmd.TableName {
				return nil, s, fmt.Errorf("unexpected table name %s", cmd.ColumnsToDelete[i])
			}
			cmd.ColumnsToDelete[i] = parts[1]
		}
	}

	// Replace all question marks with prepared params: in wheren expressions
	paramIdx := 0
	if len(cmd.WhereExpLexems) > 0 {
		for lexemIdx := range cmd.WhereExpLexems {
			if cmd.WhereExpLexems[lexemIdx].T == LexemQuestionMark {
				if paramIdx >= len(preparedQueryParams) {
					return nil, s, fmt.Errorf("not enough prepared params supplied: %d", len(preparedQueryParams))
				}
				cmd.WhereExpLexems[lexemIdx] = &Lexem{LexemPointedIdent, fmt.Sprintf("params.param%03d", paramIdx)}
				paramIdx++
			}
		}
	}

	if len(cmd.WhereExpLexems) > 0 {
		cmd.WhereExpAst, err = lexemsToAstExpr(cmd.WhereExpLexems)
		if err != nil {
			return nil, s, fmt.Errorf("cannot build ast from delete where expression: %s", err.Error())
		}
	}

	return &cmd, s, nil
}

func ParseCommands(s string, preparedQueryParams []interface{}) ([]Command, error) {
	cmds := make([]Command, 0)
	for {
		var l *Lexem
		l, s = getSemicolon(s)
		if l != nil {
			continue
		}
		if s == "" {
			break
		}
		l, s = getKeyword(s, `(?i)(CREATE\s+KEYSPACE|USE|DROP\s+KEYSPACE|CREATE\s+TABLE|TRUNCATE|DROP\s+TABLE|SELECT|INSERT\s+INTO|UPDATE|DELETE)(\b)`, false)
		if l != nil {
			var cmd Command
			var err error
			switch l.V {
			case "CREATE KEYSPACE":
				cmd, s, err = parseCreateKeyspace(s)
				if err != nil {
					return nil, fmt.Errorf("cannot parse CREATE KEYSPACE: %s", err.Error())
				}
			case "USE":
				cmd, s, err = parseUseKeyspace(s)
				if err != nil {
					return nil, fmt.Errorf("cannot parse USE KEYSPACE: %s", err.Error())
				}
			case "DROP KEYSPACE":
				cmd, s, err = parseDropKeyspace(s)
				if err != nil {
					return nil, fmt.Errorf("cannot parse DROP KEYSPACE: %s", err.Error())
				}
			case "CREATE TABLE":
				cmd, s, err = parseCreateTable(s)
				if err != nil {
					return nil, fmt.Errorf("cannot parse CREATE TABLE: %s", err.Error())
				}
			case "TRUNCATE":
				cmd, s, err = parseTruncateTable(s)
				if err != nil {
					return nil, fmt.Errorf("cannot parse TRUNCATE: %s", err.Error())
				}
			case "DROP TABLE":
				cmd, s, err = parseDropTable(s)
				if err != nil {
					return nil, fmt.Errorf("cannot parse DROP TABLE: %s", err.Error())
				}
			case "SELECT":
				cmd, s, err = parseSelect(s, preparedQueryParams)
				if err != nil {
					return nil, fmt.Errorf("cannot parse SELECT: %s", err.Error())
				}
			case "INSERT INTO":
				cmd, s, err = parseInsert(s, preparedQueryParams)
				if err != nil {
					return nil, fmt.Errorf("cannot parse INSERT: %s", err.Error())
				}
			case "UPDATE":
				cmd, s, err = parseUpdate(s, preparedQueryParams)
				if err != nil {
					return nil, fmt.Errorf("cannot parse UPDATE: %s", err.Error())
				}
			case "DELETE":
				cmd, s, err = parseDelete(s, preparedQueryParams)
				if err != nil {
					return nil, fmt.Errorf("cannot parse DELETE: %s", err.Error())
				}
			default:
				return nil, fmt.Errorf("unexpected command, dev error: %s", s)
			}
			cmds = append(cmds, cmd)
			continue
		}
		return nil, fmt.Errorf("unexpected command text, a semicolon expected: %s", s)
	}

	// Update keyspace context from "USE <keyspace>"
	curKs := ""
	for i := range len(cmds) {
		cmdUseKeyspace, ok := cmds[i].(*CommandUseKeyspace)
		if ok {
			curKs = cmdUseKeyspace.GetCtxKeyspace()
		} else {
			switch cmds[i].(type) {
			case *CommandCreateTable, *CommandTruncateTable, *CommandDropTable, *CommandSelect, *CommandInsert, *CommandUpdate, *CommandDelete:
				localCtxKs := cmds[i].GetCtxKeyspace()
				if localCtxKs == "" {
					if curKs == "" {
						return nil, fmt.Errorf("cannot detect keyspace for command %d, are you missing USE <keyspace>?", i)
					}
					cmds[i].SetCtxKeyspace(curKs)
				}
			}
		}
	}
	return cmds, nil
}
