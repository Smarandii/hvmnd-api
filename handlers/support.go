package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"hvmnd/api/db"
	"hvmnd/api/models"
	"hvmnd/api/utils"
	"net/http"
	"strconv"
	"strings"
)

// =====================
//  Support Agents
// =====================

// GET /api/v1/support/agents
// GET /api/v1/support/agents/{id}
func GetSupportAgents(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		id = r.PathValue("id")
	}
	telegramID := r.URL.Query().Get("telegram_user_id")
	isActive := r.URL.Query().Get("is_active")

	query := `SELECT id, telegram_user_id, nickname, is_active, created_at FROM support_agents WHERE 1=1`
	var args []interface{}
	idx := 1
	if id != "" {
		query += fmt.Sprintf(" AND id = $%d", idx)
		args = append(args, id)
		idx++
	}
	if telegramID != "" {
		query += fmt.Sprintf(" AND telegram_user_id = $%d", idx)
		args = append(args, telegramID)
		idx++
	}
	if isActive != "" {
		query += fmt.Sprintf(" AND is_active = $%d", idx)
		active, _ := strconv.ParseBool(isActive)
		args = append(args, active)
		idx++
	}

	rows, err := db.PostgresEngine.Query(query, args...)
	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to query support agents", Error: err.Error()})
		return
	}
	defer rows.Close()

	var agents []models.SupportAgent
	for rows.Next() {
		var a models.SupportAgent
		if err := rows.Scan(&a.ID, &a.TelegramUserID, &a.Nickname, &a.IsActive, &a.CreatedAt); err != nil {
			utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Row scan error", Error: err.Error()})
			return
		}
		agents = append(agents, a)
	}
	if len(agents) == 0 {
		utils.WriteJSONResponse(w, http.StatusNotFound, models.APIResponse{Success: false, Message: "No agents found"})
		return
	}
	utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{Success: true, Data: agents})
}

// POST /api/v1/support/agents
func CreateSupportAgent(w http.ResponseWriter, r *http.Request) {
	var in struct {
		TelegramUserID int64  `json:"telegram_user_id"`
		Nickname       string `json:"nickname"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: "Invalid JSON", Error: err.Error()})
		return
	}
	if in.TelegramUserID == 0 || in.Nickname == "" {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: "telegram_user_id and nickname are required"})
		return
	}

	var agent models.SupportAgent
	err := db.PostgresEngine.QueryRow(`INSERT INTO support_agents (telegram_user_id, nickname) VALUES ($1,$2) RETURNING id, telegram_user_id, nickname, is_active, created_at`, in.TelegramUserID, in.Nickname).
		Scan(&agent.ID, &agent.TelegramUserID, &agent.Nickname, &agent.IsActive, &agent.CreatedAt)
	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to create agent", Error: err.Error()})
		return
	}
	utils.WriteJSONResponse(w, http.StatusCreated, models.APIResponse{Success: true, Message: "Support agent created", Data: agent})
}

// PATCH /api/v1/support/agents/{id}
func UpdateSupportAgent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: "Invalid agent id"})
		return
	}
	var in struct {
		Nickname *string `json:"nickname,omitempty"`
		IsActive *bool   `json:"is_active,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: "Invalid JSON", Error: err.Error()})
		return
	}
	setParts := []string{}
	args := []interface{}{}
	idx := 1
	if in.Nickname != nil {
		setParts = append(setParts, fmt.Sprintf("nickname = $%d", idx))
		args = append(args, *in.Nickname)
		idx++
	}
	if in.IsActive != nil {
		setParts = append(setParts, fmt.Sprintf("is_active = $%d", idx))
		args = append(args, *in.IsActive)
		idx++
	}
	if len(setParts) == 0 {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: "No updatable fields provided"})
		return
	}
	args = append(args, id) // last param for WHERE
	query := fmt.Sprintf("UPDATE support_agents SET %s WHERE id = $%d RETURNING id, telegram_user_id, nickname, is_active, created_at", strings.Join(setParts, ", "), idx)

	var agent models.SupportAgent
	if err := db.PostgresEngine.QueryRow(query, args...).Scan(&agent.ID, &agent.TelegramUserID, &agent.Nickname, &agent.IsActive, &agent.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			utils.WriteJSONResponse(w, http.StatusNotFound, models.APIResponse{Success: false, Message: "Agent not found"})
			return
		}
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to update agent", Error: err.Error()})
		return
	}
	utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{Success: true, Message: "Agent updated", Data: agent})
}

// =====================
//  Support Chats
// =====================

// GET /api/v1/support/chats
func GetSupportChats(w http.ResponseWriter, r *http.Request) {
	chatID := r.URL.Query().Get("id")
	customerID := r.URL.Query().Get("customer_id")
	agentID := r.URL.Query().Get("agent_id")
	status := r.URL.Query().Get("status")

	query := `SELECT id, customer_id, agent_id, telegram_chat_id, status, opened_at, closed_at FROM support_chats WHERE 1=1`
	var args []interface{}
	idx := 1
	if chatID != "" {
		query += fmt.Sprintf(" AND id = $%d", idx)
		args = append(args, chatID)
		idx++
	}
	if customerID != "" {
		query += fmt.Sprintf(" AND customer_id = $%d", idx)
		args = append(args, customerID)
		idx++
	}
	if agentID != "" {
		query += fmt.Sprintf(" AND agent_id = $%d", idx)
		args = append(args, agentID)
		idx++
	}
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", idx)
		args = append(args, status)
		idx++
	}

	rows, err := db.PostgresEngine.Query(query, args...)
	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to query chats", Error: err.Error()})
		return
	}
	defer rows.Close()

	var chats []models.SupportChat
	for rows.Next() {
		var c models.SupportChat
		if err := rows.Scan(&c.ID, &c.CustomerID, &c.AgentID, &c.TelegramChatID, &c.Status, &c.OpenedAt, &c.ClosedAt); err != nil {
			utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Row scan error", Error: err.Error()})
			return
		}
		chats = append(chats, c)
	}
	if len(chats) == 0 {
		utils.WriteJSONResponse(w, http.StatusNotFound, models.APIResponse{Success: false, Message: "No chats found"})
		return
	}
	utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{Success: true, Data: chats})
}

// POST /api/v1/support/chats
func CreateSupportChat(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CustomerID int  `json:"customer_id"`
		AgentID    *int `json:"agent_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: "Invalid JSON", Error: err.Error()})
		return
	}
	if in.CustomerID == 0 {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: "customer_id is required"})
		return
	}
	var chat models.SupportChat
	err := db.PostgresEngine.QueryRow(`INSERT INTO support_chats (customer_id, agent_id) VALUES ($1,$2) RETURNING id, customer_id, agent_id, telegram_chat_id, status, opened_at, closed_at`, in.CustomerID, in.AgentID).
		Scan(&chat.ID, &chat.CustomerID, &chat.AgentID, &chat.TelegramChatID, &chat.Status, &chat.OpenedAt, &chat.ClosedAt)
	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to create chat", Error: err.Error()})
		return
	}
	utils.WriteJSONResponse(w, http.StatusCreated, models.APIResponse{Success: true, Message: "Chat created", Data: chat})
}

// PATCH /api/v1/support/chats/close/{id}
func CloseSupportChat(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: "Invalid chat id"})
		return
	}
	var chat models.SupportChat
	err = db.PostgresEngine.QueryRow(`UPDATE support_chats SET status=$1, closed_at=NOW() WHERE id=$2 AND status=$3 RETURNING id, customer_id, agent_id, telegram_chat_id, status, opened_at, closed_at`, models.ChatStatusClosed, id, models.ChatStatusOpen).
		Scan(&chat.ID, &chat.CustomerID, &chat.AgentID, &chat.TelegramChatID, &chat.Status, &chat.OpenedAt, &chat.ClosedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteJSONResponse(w, http.StatusNotFound, models.APIResponse{Success: false, Message: "Chat not found or already closed"})
			return
		}
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to close chat", Error: err.Error()})
		return
	}
	utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{Success: true, Message: "Chat closed", Data: chat})
}

// =====================
//  Support Messages
// =====================

// GET /api/v1/support/messages
func GetSupportMessages(w http.ResponseWriter, r *http.Request) {
	chatID := r.URL.Query().Get("chat_id")
	if chatID == "" {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: "chat_id is required"})
		return
	}
	query := `SELECT id, chat_id, sender, sender_customer_id, sender_agent_id, telegram_message_id, content, created_at, delivered_to_telegram, delivered_to_web, read_by_customer, read_by_agent FROM support_messages WHERE chat_id = $1 ORDER BY created_at`
	rows, err := db.PostgresEngine.Query(query, chatID)
	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to query messages", Error: err.Error()})
		return
	}
	defer rows.Close()

	var msgs []models.SupportMessage
	for rows.Next() {
		var m models.SupportMessage
		if err := rows.Scan(&m.ID, &m.ChatID, &m.Sender, &m.SenderCustomerID, &m.SenderAgentID, &m.TelegramMessageID, &m.Content, &m.CreatedAt, &m.DeliveredToTelegram, &m.DeliveredToWeb, &m.ReadByCustomer, &m.ReadByAgent); err != nil {
			utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Row scan error", Error: err.Error()})
			return
		}
		msgs = append(msgs, m)
	}
	if len(msgs) == 0 {
		utils.WriteJSONResponse(w, http.StatusNotFound, models.APIResponse{Success: false, Message: "No messages found"})
		return
	}
	utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{Success: true, Data: msgs})
}

// POST /api/v1/support/messages
func CreateSupportMessage(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ChatID           int    `json:"chat_id"`
		Sender           string `json:"sender"` // "customer" | "agent" | "system"
		Content          string `json:"content"`
		SenderCustomerID *int   `json:"sender_customer_id,omitempty"`
		SenderAgentID    *int   `json:"sender_agent_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: "Invalid JSON", Error: err.Error()})
		return
	}
	if in.ChatID == 0 || in.Content == "" || (in.Sender != models.SenderAgent && in.Sender != models.SenderCustomer && in.Sender != models.SenderSystem) {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: "chat_id, sender and content are required with valid sender"})
		return
	}

	// basic sender ownership check can be added here

	var msg models.SupportMessage
	err := db.PostgresEngine.QueryRow(`
        INSERT INTO support_messages (chat_id, sender, sender_customer_id, sender_agent_id, content)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id, chat_id, sender, sender_customer_id, sender_agent_id, telegram_message_id, content, created_at, delivered_to_telegram, delivered_to_web, read_by_customer, read_by_agent`,
		in.ChatID, in.Sender, in.SenderCustomerID, in.SenderAgentID, in.Content,
	).Scan(&msg.ID, &msg.ChatID, &msg.Sender, &msg.SenderCustomerID, &msg.SenderAgentID, &msg.TelegramMessageID, &msg.Content, &msg.CreatedAt, &msg.DeliveredToTelegram, &msg.DeliveredToWeb, &msg.ReadByCustomer, &msg.ReadByAgent)
	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to save message", Error: err.Error()})
		return
	}
	utils.WriteJSONResponse(w, http.StatusCreated, models.APIResponse{Success: true, Message: "Message queued", Data: msg})
}
