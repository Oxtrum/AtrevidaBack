// Runner temporal: go run ./cmd/_sql "<sql>"  |  go run ./cmd/_sql -f archivo.sql
// SELECT/WITH/SHOW imprimen filas como JSON; lo demas se ejecuta tal cual.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"atrevida-agenda-api/config"
	"atrevida-agenda-api/db"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("uso: _sql \"<sql>\" | -f archivo.sql")
	}
	query := os.Args[1]
	if query == "-f" {
		b, err := os.ReadFile(os.Args[2])
		if err != nil {
			log.Fatal(err)
		}
		query = string(b)
	}
	config.Load()
	pg, err := db.Connect(config.App)
	if err != nil {
		log.Fatal(err)
	}
	defer pg.Close()

	head := strings.ToUpper(strings.TrimSpace(query))
	if strings.HasPrefix(head, "SELECT") || strings.HasPrefix(head, "WITH") || strings.HasPrefix(head, "SHOW") {
		rows, err := pg.Queryx(query)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			m := map[string]any{}
			if err := rows.MapScan(m); err != nil {
				log.Fatal(err)
			}
			for k, v := range m {
				if b, ok := v.([]byte); ok {
					m[k] = string(b)
				}
			}
			out, _ := json.Marshal(m)
			fmt.Println(string(out))
		}
		return
	}
	res, err := pg.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
	n, _ := res.RowsAffected()
	fmt.Println("ok, filas afectadas:", n)
}
