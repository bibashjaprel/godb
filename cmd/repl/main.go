// Command repl is an interactive shell for godb: a tiny database
// engine built from scratch (in-memory storage + hash indexing + a
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"godb/engine"
)

func main() {
	eng := engine.New()
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("godb — a toy database engine. Type SQL-like statements ending in ';'.")
	fmt.Println(`Try: CREATE TABLE users (id INT PRIMARY KEY, name TEXT, active BOOL);`)
	fmt.Println(`Meta commands: \dt (list tables), \q (quit)`)
	fmt.Println()

	var buf strings.Builder

	prompt := func() {
		if buf.Len() == 0 {
			fmt.Print("godb> ")
		} else {
			fmt.Print("   -> ")
		}
	}

	prompt()
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if buf.Len() == 0 {
			switch trimmed {
			case `\q`, "exit", "quit":
				return
			case `\dt`:
				printTables(eng)
				prompt()
				continue
			case "":
				prompt()
				continue
			}
		}

		buf.WriteString(line)
		buf.WriteString(" ")

		if strings.HasSuffix(trimmed, ";") {
			stmt := buf.String()
			buf.Reset()
			runStatement(eng, stmt)
		}
		prompt()
	}
	fmt.Println()
}

func printTables(eng *engine.Engine) {
	names := eng.DB.TableNames()
	if len(names) == 0 {
		fmt.Println("(no tables)")
		return
	}
	for _, n := range names {
		fmt.Println(" ", n)
	}
}

func runStatement(eng *engine.Engine, stmt string) {
	res, err := eng.Execute(stmt)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	if res.Message != "" {
		fmt.Println(res.Message)
		return
	}
	printTable(res.Columns, res.Rows)
}

func printTable(columns []string, rows [][]interface{}) {
	if len(columns) == 0 {
		fmt.Println("(no columns)")
		return
	}

	widths := make([]int, len(columns))
	cellStrs := make([][]string, len(rows))
	for i, c := range columns {
		widths[i] = len(c)
	}
	for ri, row := range rows {
		cellStrs[ri] = make([]string, len(row))
		for ci, v := range row {
			s := formatValue(v)
			cellStrs[ri][ci] = s
			if len(s) > widths[ci] {
				widths[ci] = len(s)
			}
		}
	}

	printRow := func(cells []string) {
		parts := make([]string, len(cells))
		for i, c := range cells {
			parts[i] = padRight(c, widths[i])
		}
		fmt.Println(" " + strings.Join(parts, " | "))
	}

	printRow(columns)
	sepParts := make([]string, len(widths))
	for i, w := range widths {
		sepParts[i] = strings.Repeat("-", w)
	}
	fmt.Println(" " + strings.Join(sepParts, "-+-"))

	for _, cells := range cellStrs {
		printRow(cells)
	}
	fmt.Printf("(%d row(s))\n", len(rows))
}

func formatValue(v interface{}) string {
	switch x := v.(type) {
	case int64:
		return strconv.FormatInt(x, 10)
	case string:
		return x
	case bool:
		return strconv.FormatBool(x)
	case nil:
		return "NULL"
	default:
		return fmt.Sprintf("%v", x)
	}
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}
