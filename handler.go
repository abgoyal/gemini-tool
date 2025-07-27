package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type Handler struct {
	DB     *sql.DB
	APIKey string
}

func NewHandler(db *sql.DB, apiKey string) *Handler {
	return &Handler{DB: db, APIKey: apiKey}
}

func isUniqueConstraintError(err error) bool {
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func (h *Handler) PromptsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		prompts, err := GetPrompts(h.DB)
		if err != nil {
			http.Error(w, `{"error":"Failed to get prompts"}`, http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(prompts)

	case http.MethodPost:
		var p Prompt
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
			return
		}
		id, err := CreatePrompt(h.DB, &p)
		if err != nil {
			if isUniqueConstraintError(err) {
				http.Error(w, `{"error":"A prompt with this name already exists."}`, http.StatusConflict)
			} else {
				http.Error(w, `{"error":"Failed to create prompt"}`, http.StatusInternalServerError)
			}
			return
		}
		p.ID = id
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(p)

	case http.MethodPut:
		var p Prompt
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
			return
		}
		if p.ID == 0 {
			http.Error(w, `{"error":"Prompt ID is required for update"}`, http.StatusBadRequest)
			return
		}
		err := UpdatePrompt(h.DB, &p)
		if err != nil {
			if isUniqueConstraintError(err) {
				http.Error(w, `{"error":"A prompt with this name already exists."}`, http.StatusConflict)
			} else {
				http.Error(w, `{"error":"Failed to update prompt"}`, http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(p)

	default:
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

type GenerateRequest struct {
	PromptID  int64  `json:"prompt_id"`
	UserInput string `json:"user_input"`
	Model     string `json:"model,omitempty"`
}

func (h *Handler) GenerateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	var p Prompt
	err := h.DB.QueryRow("SELECT id, name, prompt_text, separator, model FROM prompts WHERE id = ?", req.PromptID).Scan(&p.ID, &p.Name, &p.PromptText, &p.Separator, &p.Model)
	if err != nil {
		http.Error(w, `{"error":"Prompt not found"}`, http.StatusNotFound)
		return
	}

	modelToUse := p.Model
	if req.Model != "" {
		modelToUse = req.Model
	}

	chat := Chat{
		PromptID:         sql.NullInt64{Int64: p.ID, Valid: true},
		UserInput:        req.UserInput,
		RequestTimestamp: time.Now().UTC(),
		PromptName:       sql.NullString{String: p.Name, Valid: true},
		ModelUsed:        sql.NullString{String: modelToUse, Valid: true},
	}
	chatID, err := CreateChat(h.DB, &chat)
	if err != nil {
		http.Error(w, `{"error":"Failed to log chat request"}`, http.StatusInternalServerError)
		return
	}
	chat.ID = chatID

	startTime := time.Now()
	combinedPrompt := fmt.Sprintf("%s\n%s\n%s", p.PromptText, p.Separator, req.UserInput)

	result, err := GenerateText(h.APIKey, modelToUse, combinedPrompt)

	chat.ResponseTimestamp = sql.NullTime{Time: time.Now().UTC(), Valid: true}
	chat.TimeTakenMs = sql.NullInt64{Int64: time.Since(startTime).Milliseconds(), Valid: true}

	if err != nil {
		chat.ErrorMessage = sql.NullString{String: err.Error(), Valid: true}
		UpdateChatOnCompletion(h.DB, &chat)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	chat.ModelOutput = sql.NullString{String: result.Text, Valid: true}
	chat.InputTokenCount = sql.NullInt64{Int64: int64(result.InputTokenCount), Valid: true}
	chat.OutputTokenCount = sql.NullInt64{Int64: int64(result.OutputTokenCount), Valid: true}

	if err := UpdateChatOnCompletion(h.DB, &chat); err != nil {
		log.Printf("Failed to update chat log %d: %v", chat.ID, err)
	}

	json.NewEncoder(w).Encode(chat)
}

func (h *Handler) ChatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	chats, err := GetChats(h.DB)
	if err != nil {
		http.Error(w, `{"error":"Failed to get chats"}`, http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(chats)
}

func (h *Handler) ModelsHandler(w http.ResponseWriter, r *http.Request) {
	models, err := ListModels(h.APIKey)
	if err != nil {
		log.Printf("ERROR: Failed to fetch models from API: %v", err)
		http.Error(w, `{"error":"Failed to fetch models from Google API"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("--> %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("<-- %s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
