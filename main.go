package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

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
		ID string `json:"id"`
	} `json:"sys"`
	Fields struct {
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"createdAt"`
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
}

func syncData(cmd *cobra.Command, args []string) {
	entries := []string{
		"6QRk7gQYmOyJ1eMG9H4jbB",
		"41RUO5w4oIpNuwaqHuSwEc",
		"4Li6w5uVbJNVXYVxWjWVoZ",
	}

	connStr := "user=postgres dbname=postgres sslmode=disable password=tkz2001r"
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

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

	return &entry, nil
}

func saveEntry(entry *Entry) error {
	_, err := db.Exec("INSERT INTO entries (id, name, created_at) VALUES ($1, $2, $3)", entry.Sys.ID, entry.Fields.Name, entry.Fields.CreatedAt)
	return err
}
