package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Prompt struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	PromptText string `json:"prompt_text"`
	Separator  string `json:"separator"`
	Model      string `json:"model"`
}

type Chat struct {
	ID                int64          `json:"id"`
	PromptID          sql.NullInt64  `json:"prompt_id"`
	UserInput         string         `json:"user_input"`
	ModelOutput       sql.NullString `json:"model_output"`
	RequestTimestamp  time.Time      `json:"request_timestamp"`
	ResponseTimestamp sql.NullTime   `json:"response_timestamp"`
	TimeTakenMs       sql.NullInt64  `json:"time_taken_ms"`
	InputTokenCount   sql.NullInt64  `json:"input_token_count"`
	OutputTokenCount  sql.NullInt64  `json:"output_token_count"`
	ErrorMessage      sql.NullString `json:"error_message"`
	PromptName        sql.NullString `json:"prompt_name"`
	ModelUsed         sql.NullString `json:"model_used"`
}

func migrateDB(db *sql.DB) error {
	log.Println("Checking database schema...")

	rows, err := db.Query("PRAGMA table_info(chats)")
	if err != nil {
		return fmt.Errorf("failed to query chats table info: %w", err)
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var (
			cid        int
			name       string
			ctype      string
			notnull    bool
			dflt_value sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt_value, &pk); err != nil {
			return fmt.Errorf("failed to scan table info row: %w", err)
		}
		columns[name] = true
	}

	if !columns["prompt_name"] {
		log.Println("Schema migration: adding 'prompt_name' to 'chats' table.")
		if _, err := db.Exec("ALTER TABLE chats ADD COLUMN prompt_name TEXT"); err != nil {
			return fmt.Errorf("failed to add 'prompt_name' column: %w", err)
		}
	}

	if !columns["model_used"] {
		log.Println("Schema migration: adding 'model_used' to 'chats' table.")
		if _, err := db.Exec("ALTER TABLE chats ADD COLUMN model_used TEXT"); err != nil {
			return fmt.Errorf("failed to add 'model_used' column: %w", err)
		}
	}

	log.Println("Database schema is up to date.")
	return nil
}

func backfillMissingChatInfo(db *sql.DB) error {
	log.Println("Checking for historical chat data to backfill...")
	updateSQL := `
	UPDATE chats 
	SET 
		prompt_name = (SELECT name FROM prompts WHERE prompts.id = chats.prompt_id),
		model_used = (SELECT model FROM prompts WHERE prompts.id = chats.prompt_id)
	WHERE prompt_name IS NULL AND prompt_id IS NOT NULL;`

	result, err := db.Exec(updateSQL)
	if err != nil {
		return fmt.Errorf("failed to execute chat info backfill: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected during backfill: %w", err)
	}
	if rowsAffected > 0 {
		log.Printf("Successfully backfilled historical data for %d chat records.", rowsAffected)
	} else {
		log.Println("No historical data needed backfilling.")
	}
	return nil
}

func InitDB(filepath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		return nil, err
	}

	createPromptsTable := `CREATE TABLE IF NOT EXISTS prompts (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE, prompt_text TEXT NOT NULL, separator TEXT NOT NULL DEFAULT '---', model TEXT NOT NULL);`
	if _, err := db.Exec(createPromptsTable); err != nil {
		return nil, err
	}

	createChatsTable := `CREATE TABLE IF NOT EXISTS chats (id INTEGER PRIMARY KEY AUTOINCREMENT, prompt_id INTEGER, user_input TEXT NOT NULL, model_output TEXT, request_timestamp DATETIME NOT NULL, response_timestamp DATETIME, time_taken_ms INTEGER, input_token_count INTEGER, output_token_count INTEGER, error_message TEXT, FOREIGN KEY(prompt_id) REFERENCES prompts(id));`
	if _, err := db.Exec(createChatsTable); err != nil {
		return nil, err
	}

	if err := migrateDB(db); err != nil {
		return nil, fmt.Errorf("schema migration failed: %w", err)
	}
	if err := backfillMissingChatInfo(db); err != nil {
		return nil, fmt.Errorf("data backfill failed: %w", err)
	}
	return db, nil
}

func CreatePrompt(db *sql.DB, p *Prompt) (int64, error) {
	res, err := db.Exec("INSERT INTO prompts (name, prompt_text, separator, model) VALUES (?, ?, ?, ?)", p.Name, p.PromptText, p.Separator, p.Model)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func GetPrompts(db *sql.DB) ([]Prompt, error) {
	rows, err := db.Query("SELECT id, name, prompt_text, separator, model FROM prompts ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var prompts []Prompt
	for rows.Next() {
		var p Prompt
		if err := rows.Scan(&p.ID, &p.Name, &p.PromptText, &p.Separator, &p.Model); err != nil {
			return nil, err
		}
		prompts = append(prompts, p)
	}
	return prompts, nil
}

func UpdatePrompt(db *sql.DB, p *Prompt) error {
	_, err := db.Exec("UPDATE prompts SET name = ?, prompt_text = ?, separator = ?, model = ? WHERE id = ?", p.Name, p.PromptText, p.Separator, p.Model, p.ID)
	return err
}

func CreateChat(db *sql.DB, c *Chat) (int64, error) {
	res, err := db.Exec("INSERT INTO chats (prompt_id, user_input, request_timestamp, prompt_name, model_used) VALUES (?, ?, ?, ?, ?)", c.PromptID, c.UserInput, c.RequestTimestamp, c.PromptName, c.ModelUsed)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func UpdateChatOnCompletion(db *sql.DB, c *Chat) error {
	_, err := db.Exec(`
		UPDATE chats SET 
			model_output = ?, response_timestamp = ?, time_taken_ms = ?, 
			input_token_count = ?, output_token_count = ?, error_message = ? 
		WHERE id = ?`,
		c.ModelOutput, c.ResponseTimestamp, c.TimeTakenMs, c.InputTokenCount, c.OutputTokenCount, c.ErrorMessage, c.ID)
	return err
}

func GetChats(db *sql.DB) ([]Chat, error) {
	query := `
	SELECT 
		id, prompt_id, user_input, model_output, request_timestamp, 
		response_timestamp, time_taken_ms, input_token_count, output_token_count, 
		error_message, prompt_name, model_used 
	FROM chats ORDER BY request_timestamp DESC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []Chat
	for rows.Next() {
		var c Chat
		err := rows.Scan(&c.ID, &c.PromptID, &c.UserInput, &c.ModelOutput, &c.RequestTimestamp,
			&c.ResponseTimestamp, &c.TimeTakenMs, &c.InputTokenCount, &c.OutputTokenCount,
			&c.ErrorMessage, &c.PromptName, &c.ModelUsed)
		if err != nil {
			return nil, err
		}
		chats = append(chats, c)
	}
	return chats, nil
}
