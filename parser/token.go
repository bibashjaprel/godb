package parser

// TokenType identifies the lexical class of a Token.
type TokenType int

const (
	EOF TokenType = iota
	IDENT
	NUMBER
	STRING

	// Keywords
	CREATE
	TABLE
	INSERT
	INTO
	VALUES
	SELECT
	FROM
	WHERE
	UPDATE
	SET
	DELETE
	AND
	PRIMARY
	KEY
	INDEX
	ON
	TRUE
	FALSE
	INT_TYPE
	TEXT_TYPE
	BOOL_TYPE

	// Symbols
	LPAREN
	RPAREN
	COMMA
	SEMICOLON
	STAR
	EQ
	NEQ
	LT
	LE
	GT
	GE
)

var keywords = map[string]TokenType{
	"CREATE":  CREATE,
	"TABLE":   TABLE,
	"INSERT":  INSERT,
	"INTO":    INTO,
	"VALUES":  VALUES,
	"SELECT":  SELECT,
	"FROM":    FROM,
	"WHERE":   WHERE,
	"UPDATE":  UPDATE,
	"SET":     SET,
	"DELETE":  DELETE,
	"AND":     AND,
	"PRIMARY": PRIMARY,
	"KEY":     KEY,
	"INDEX":   INDEX,
	"ON":      ON,
	"TRUE":    TRUE,
	"FALSE":   FALSE,
	"INT":     INT_TYPE,
	"INTEGER": INT_TYPE,
	"TEXT":    TEXT_TYPE,
	"VARCHAR": TEXT_TYPE,
	"STRING":  TEXT_TYPE,
	"BOOL":    BOOL_TYPE,
	"BOOLEAN": BOOL_TYPE,
}

// Token is a single lexical token: its class, and the literal text
// that produced it (for IDENT/NUMBER/STRING).
type Token struct {
	Type    TokenType
	Literal string
	Pos     int
}

func (t Token) String() string {
	return t.Literal
}
