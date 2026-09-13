package parser

import (
	"fmt"
	"strconv"
	"strings"
)

// Parser turns a token stream into a single Statement.
type Parser struct {
	tokens []Token
	pos    int
}

// Parse tokenizes and parses a single SQL-like statement. A trailing
// semicolon is optional.
func Parse(input string) (Statement, error) {
	tokens, err := Tokenize(input)
	if err != nil {
		return nil, err
	}
	p := &Parser{tokens: tokens}
	return p.parseStatement()
}

func (p *Parser) cur() Token  { return p.tokens[p.pos] }
func (p *Parser) advance() Token {
	t := p.tokens[p.pos]
	if p.pos < len(p.tokens)-1 {
		p.pos++
	}
	return t
}

func (p *Parser) expect(tt TokenType, what string) (Token, error) {
	if p.cur().Type != tt {
		return Token{}, fmt.Errorf("expected %s but got %q", what, p.cur().Literal)
	}
	return p.advance(), nil
}

func (p *Parser) parseStatement() (Statement, error) {
	switch p.cur().Type {
	case CREATE:
		return p.parseCreate()
	case INSERT:
		return p.parseInsert()
	case SELECT:
		return p.parseSelect()
	case UPDATE:
		return p.parseUpdate()
	case DELETE:
		return p.parseDelete()
	case EOF:
		return nil, fmt.Errorf("empty statement")
	default:
		return nil, fmt.Errorf("unexpected token %q at start of statement", p.cur().Literal)
	}
}

// ---- CREATE TABLE / CREATE INDEX ----

func (p *Parser) parseCreate() (Statement, error) {
	p.advance() // CREATE
	switch p.cur().Type {
	case TABLE:
		return p.parseCreateTable()
	case INDEX:
		return p.parseCreateIndex()
	default:
		return nil, fmt.Errorf("expected TABLE or INDEX after CREATE, got %q", p.cur().Literal)
	}
}

func (p *Parser) parseCreateTable() (Statement, error) {
	p.advance() // TABLE
	nameTok, err := p.expect(IDENT, "table name")
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(LPAREN, "'('"); err != nil {
		return nil, err
	}

	var cols []ColumnDef
	for {
		colTok, err := p.expect(IDENT, "column name")
		if err != nil {
			return nil, err
		}
		typTok := p.advance()
		var typName string
		switch typTok.Type {
		case INT_TYPE:
			typName = "INT"
		case TEXT_TYPE:
			typName = "TEXT"
		case BOOL_TYPE:
			typName = "BOOL"
		default:
			return nil, fmt.Errorf("expected a column type (INT/TEXT/BOOL) for column %q, got %q", colTok.Literal, typTok.Literal)
		}
		def := ColumnDef{Name: colTok.Literal, Type: typName}
		if p.cur().Type == PRIMARY {
			p.advance()
			if _, err := p.expect(KEY, "KEY"); err != nil {
				return nil, err
			}
			def.PrimaryKey = true
		}
		cols = append(cols, def)

		if p.cur().Type == COMMA {
			p.advance()
			continue
		}
		break
	}
	if _, err := p.expect(RPAREN, "')'"); err != nil {
		return nil, err
	}
	p.consumeOptionalSemicolon()
	return CreateTableStmt{Table: nameTok.Literal, Columns: cols}, nil
}

func (p *Parser) parseCreateIndex() (Statement, error) {
	p.advance() // INDEX
	if _, err := p.expect(ON, "ON"); err != nil {
		return nil, err
	}
	tableTok, err := p.expect(IDENT, "table name")
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(LPAREN, "'('"); err != nil {
		return nil, err
	}
	colTok, err := p.expect(IDENT, "column name")
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(RPAREN, "')'"); err != nil {
		return nil, err
	}
	p.consumeOptionalSemicolon()
	return CreateIndexStmt{Table: tableTok.Literal, Column: colTok.Literal}, nil
}

// ---- INSERT ----

func (p *Parser) parseInsert() (Statement, error) {
	p.advance() // INSERT
	if _, err := p.expect(INTO, "INTO"); err != nil {
		return nil, err
	}
	tableTok, err := p.expect(IDENT, "table name")
	if err != nil {
		return nil, err
	}

	var columns []string
	if p.cur().Type == LPAREN {
		p.advance()
		for {
			colTok, err := p.expect(IDENT, "column name")
			if err != nil {
				return nil, err
			}
			columns = append(columns, colTok.Literal)
			if p.cur().Type == COMMA {
				p.advance()
				continue
			}
			break
		}
		if _, err := p.expect(RPAREN, "')'"); err != nil {
			return nil, err
		}
	}

	if _, err := p.expect(VALUES, "VALUES"); err != nil {
		return nil, err
	}

	var rows [][]Literal
	for {
		if _, err := p.expect(LPAREN, "'('"); err != nil {
			return nil, err
		}
		var vals []Literal
		for {
			lit, err := p.parseLiteral()
			if err != nil {
				return nil, err
			}
			vals = append(vals, lit)
			if p.cur().Type == COMMA {
				p.advance()
				continue
			}
			break
		}
		if _, err := p.expect(RPAREN, "')'"); err != nil {
			return nil, err
		}
		rows = append(rows, vals)

		if p.cur().Type == COMMA {
			p.advance()
			continue
		}
		break
	}
	p.consumeOptionalSemicolon()
	return InsertStmt{Table: tableTok.Literal, Columns: columns, Rows: rows}, nil
}

// ---- SELECT ----

func (p *Parser) parseSelect() (Statement, error) {
	p.advance() // SELECT

	var columns []string
	if p.cur().Type == STAR {
		p.advance()
		columns = []string{"*"}
	} else {
		for {
			colTok, err := p.expect(IDENT, "column name")
			if err != nil {
				return nil, err
			}
			columns = append(columns, colTok.Literal)
			if p.cur().Type == COMMA {
				p.advance()
				continue
			}
			break
		}
	}

	if _, err := p.expect(FROM, "FROM"); err != nil {
		return nil, err
	}
	tableTok, err := p.expect(IDENT, "table name")
	if err != nil {
		return nil, err
	}

	var where []Condition
	if p.cur().Type == WHERE {
		p.advance()
		where, err = p.parseConditions()
		if err != nil {
			return nil, err
		}
	}
	p.consumeOptionalSemicolon()
	return SelectStmt{Table: tableTok.Literal, Columns: columns, Where: where}, nil
}

// ---- UPDATE ----

func (p *Parser) parseUpdate() (Statement, error) {
	p.advance() // UPDATE
	tableTok, err := p.expect(IDENT, "table name")
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(SET, "SET"); err != nil {
		return nil, err
	}
	set := make(map[string]Literal)
	for {
		colTok, err := p.expect(IDENT, "column name")
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(EQ, "'='"); err != nil {
			return nil, err
		}
		lit, err := p.parseLiteral()
		if err != nil {
			return nil, err
		}
		set[colTok.Literal] = lit
		if p.cur().Type == COMMA {
			p.advance()
			continue
		}
		break
	}

	var where []Condition
	if p.cur().Type == WHERE {
		p.advance()
		where, err = p.parseConditions()
		if err != nil {
			return nil, err
		}
	}
	p.consumeOptionalSemicolon()
	return UpdateStmt{Table: tableTok.Literal, Set: set, Where: where}, nil
}

// ---- DELETE ----

func (p *Parser) parseDelete() (Statement, error) {
	p.advance() // DELETE
	if _, err := p.expect(FROM, "FROM"); err != nil {
		return nil, err
	}
	tableTok, err := p.expect(IDENT, "table name")
	if err != nil {
		return nil, err
	}
	var where []Condition
	if p.cur().Type == WHERE {
		p.advance()
		where, err = p.parseConditions()
		if err != nil {
			return nil, err
		}
	}
	p.consumeOptionalSemicolon()
	return DeleteStmt{Table: tableTok.Literal, Where: where}, nil
}

// ---- shared helpers ----

func (p *Parser) parseConditions() ([]Condition, error) {
	var conds []Condition
	for {
		colTok, err := p.expect(IDENT, "column name")
		if err != nil {
			return nil, err
		}
		opTok := p.advance()
		switch opTok.Type {
		case EQ, NEQ, LT, LE, GT, GE:
		default:
			return nil, fmt.Errorf("expected a comparison operator, got %q", opTok.Literal)
		}
		lit, err := p.parseLiteral()
		if err != nil {
			return nil, err
		}
		conds = append(conds, Condition{Column: colTok.Literal, Op: opTok.Type, Value: lit})

		if p.cur().Type == AND {
			p.advance()
			continue
		}
		break
	}
	return conds, nil
}

func (p *Parser) parseLiteral() (Literal, error) {
	tok := p.advance()
	switch tok.Type {
	case NUMBER:
		if strings.Contains(tok.Literal, ".") {
			return Literal{}, fmt.Errorf("floating point numbers are not supported, got %q", tok.Literal)
		}
		n, err := strconv.ParseInt(tok.Literal, 10, 64)
		if err != nil {
			return Literal{}, fmt.Errorf("invalid integer literal %q: %w", tok.Literal, err)
		}
		return Literal{Value: n}, nil
	case STRING:
		return Literal{Value: tok.Literal}, nil
	case TRUE:
		return Literal{Value: true}, nil
	case FALSE:
		return Literal{Value: false}, nil
	default:
		return Literal{}, fmt.Errorf("expected a literal value, got %q", tok.Literal)
	}
}

func (p *Parser) consumeOptionalSemicolon() {
	if p.cur().Type == SEMICOLON {
		p.advance()
	}
}
