package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
)

var (
	db          *sql.DB
	apiURL      = "https://cdn.contentful.com/spaces/2vskphwbz4oc/entries"
	accessToken = ""
)

type Entry struct {
	Sys struct {
		ID        string `json:"id"`
		CreatedAt string `json:"createdAt"`
	} `json:"sys"`
	Fields struct {
		Name string `json:"name"`
	} `json:"fields"`
}

func main() {
	var rootCmd = &cobra.Command{Use: "cli"}
	var syncCmd = &cobra.Command{
		Use:   "sync",
		Short: "Sync data from Contentful to PostgreSQL",
		Run:   syncData,
	}

	rootCmd.AddCommand(syncCmd)
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}

	// GraphQLスキーマの定義
	var schema, _ = graphql.NewSchema(
		graphql.SchemaConfig{
			Query: rootQuery,
		},
	)

	// GraphQLハンドラの作成
	h := handler.New(&handler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	})
	// GraphQLエンドポイントの設定
	http.Handle("/graphql", h)

	// サーバを起動
	fmt.Println("GraphQL server is running on http://localhost:8080/graphql")
	http.ListenAndServe(":8080", nil)
}

var entryType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Entry",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					entry, _ := p.Source.(*Entry)
					return entry.Sys.ID, nil
				},
			},
			"name": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					entry, _ := p.Source.(*Entry)
					return entry.Fields.Name, nil
				},
			},
			"createdAt": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					entry, _ := p.Source.(*Entry)
					return entry.Sys.CreatedAt, nil
				},
			},
		},
	},
)

var rootQuery = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"entries": &graphql.Field{
				Type: graphql.NewList(entryType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					// データベースからエントリー一覧を取得する
					fmt.Println("Fetching entries from the database...")
					rows, err := db.Query("SELECT id, name, created_at FROM entries")
					if err != nil {
						log.Println("Error querying entries:", err)
						return nil, err
					}
					defer rows.Close()

					var entries []*Entry
					for rows.Next() {
						var entry Entry
						err := rows.Scan(&entry.Sys.ID, &entry.Fields.Name, &entry.Sys.CreatedAt)
						if err != nil {
							log.Println("Error scanning entry row:", err)
							return nil, err
						}
						entries = append(entries, &entry)
					}
					fmt.Printf("Fetched %d entries from the database.\n", len(entries))
					return entries, nil
				},
			},
			"entry": &graphql.Field{
				Type: entryType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					id, ok := p.Args["id"].(string)
					if !ok {
						return nil, fmt.Errorf("ID argument is required")
					}

					// データベースから指定されたIDのエントリーを取得する
					entry := &Entry{}
					err := db.QueryRow("SELECT id, name, created_at FROM entries WHERE id = $1", id).
						Scan(&entry.Sys.ID, &entry.Fields.Name, &entry.Sys.CreatedAt)
					if err != nil {
						log.Println("Error retrieving entry:", err)
						return nil, err
					}

					return entry, nil
				},
			},
		},
	},
)

func initDB(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// テーブルが存在しない場合に自動的に作成する
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS entries (
			id          TEXT PRIMARY KEY,
			name        TEXT,
			created_at  TIMESTAMP WITH TIME ZONE
		)
	`)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func syncData(cmd *cobra.Command, args []string) {
	entries := []string{
		"6QRk7gQYmOyJ1eMG9H4jbB",
		"41RUO5w4oIpNuwaqHuSwEc",
		"4Li6w5uVbJNVXYVxWjWVoZ",
	}

	connStr := "user=postgres dbname=postgres sslmode=disable password=tkz2001r"
	var err error
	db, err = initDB(connStr)
	if err != nil {
		log.Fatal(err)
	}
	// defer db.Close()

	for _, entryID := range entries {
		entry, err := getEntry(entryID)
		if err != nil {
			log.Println("Failed to get entry:", err)
			continue
		}

		exists, err := checkEntryExists(entry.Sys.ID)
		if err != nil {
			log.Println("Failed to check entry existence:", err)
			continue
		}

		if exists {
			log.Printf("Entry with id %s already exists, skipping.", entry.Sys.ID)
			continue
		}

		err = saveEntry(entry)
		if err != nil {
			log.Println("Failed to save entry:", err)
			continue
		}
	}
}

func checkEntryExists(entryID string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM entries WHERE id = $1", entryID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func getEntry(entryID string) (*Entry, error) {
	client := &http.Client{}

	url := fmt.Sprintf("%s/%s?access_token=%s", apiURL, entryID, accessToken)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var entry Entry
	err = json.NewDecoder(resp.Body).Decode(&entry)
	if err != nil {
		return nil, err
	}
	fmt.Println("entry.Sys:", entry.Sys)
	fmt.Println("entry.Fields:", entry.Fields)

	// "createdAt" フィールドのパース
	createdAtStr := entry.Sys.CreatedAt
	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, err
	}
	entry.Sys.CreatedAt = createdAt.Format("2006-01-02 15:04:05-07:00")

	return &entry, nil
}

func saveEntry(entry *Entry) error {
	_, err := db.Exec("INSERT INTO entries (id, name, created_at) VALUES ($1, $2, $3)", entry.Sys.ID, entry.Fields.Name, entry.Sys.CreatedAt)
	return err
}
