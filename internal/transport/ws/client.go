package ws

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"imposter/internal/app"
	"imposter/internal/domain"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 4096

	// Size of the send channel buffer
	sendBufferSize = 256
)

// Client represents a WebSocket client connection
type Client struct {
	conn     *websocket.Conn
	session  *app.GameSession
	playerID string
	send     chan []byte
	done     chan struct{}
	logger   *slog.Logger
	mu       sync.Mutex
	closed   bool
}

// NewClient creates a new WebSocket client
func NewClient(conn *websocket.Conn, session *app.GameSession, playerID string, logger *slog.Logger) *Client {
	return &Client{
		conn:     conn,
		session:  session,
		playerID: playerID,
		send:     make(chan []byte, sendBufferSize),
		done:     make(chan struct{}),
		logger:   logger,
	}
}

// GetPlayerID returns the player ID for this client
func (c *Client) GetPlayerID() string {
	return c.playerID
}

// Send implements app.ClientConnection interface
func (c *Client) Send(message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	select {
	case c.send <- data:
		return nil
	default:
		// Buffer full, message dropped
		c.logger.Warn("send buffer full, message dropped", "playerID", c.playerID)
		return nil
	}
}

// Close implements app.ClientConnection interface
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	close(c.done)
	return c.conn.Close()
}

// Run starts the client's read and write pumps
func (c *Client) Run() {
	go c.writePump()
	c.readPump()
}

// readPump pumps messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.session.UnregisterClient(c.playerID)
		c.session.DisconnectPlayer(c.playerID)
		c.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Debug("websocket read error", "error", err)
			}
			break
		}

		c.handleMessage(message)
	}
}

// writePump pumps messages from the send channel to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case <-c.done:
			return
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes an incoming message from the client
func (c *Client) handleMessage(data []byte) {
	var msg ClientMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError(ErrCodeInvalidMessage, "Invalid message format")
		return
	}

	switch msg.Type {
	case MsgJoinLobby:
		c.handleJoinLobby(msg.Payload)
	case MsgStartGame:
		c.handleStartGame()
	case MsgSubmitWord:
		c.handleSubmitWord(msg.Payload)
	case MsgCastVote:
		c.handleCastVote(msg.Payload)
	case MsgRequestNewRound:
		c.handleRequestNewRound()
	case MsgPing:
		c.sendPong()
	default:
		c.sendError(ErrCodeInvalidMessage, "Unknown message type")
	}
}

// handleJoinLobby handles a join_lobby message
func (c *Client) handleJoinLobby(payload interface{}) {
	payloadMap, ok := payload.(map[string]interface{})
	if !ok {
		c.sendError(ErrCodeInvalidMessage, "Invalid payload")
		return
	}

	nickname, ok := payloadMap["nickname"].(string)
	if !ok || nickname == "" {
		c.sendError(ErrCodeInvalidMessage, "Nickname is required")
		return
	}

	// Try to add player to game
	_, err := c.session.AddPlayer(c.playerID, nickname)
	if err != nil {
		switch err {
		case domain.ErrGameFull:
			c.sendError(ErrCodeGameFull, "Game is full")
		case domain.ErrGameAlreadyStarted:
			c.sendError(ErrCodeInvalidAction, "Game has already started")
		default:
			c.sendError(ErrCodeInternalError, err.Error())
		}
		return
	}

	// Send connected confirmation
	c.sendConnected()
}

// handleStartGame handles a start_game message
func (c *Client) handleStartGame() {
	err := c.session.StartGame(c.playerID)
	if err != nil {
		switch err {
		case domain.ErrNotHost:
			c.sendError(ErrCodeNotHost, "Only the host can start the game")
		case domain.ErrNotEnoughPlayers:
			c.sendError(ErrCodeInvalidAction, "Not enough players to start")
		default:
			c.sendError(ErrCodeInternalError, err.Error())
		}
		return
	}
}

// handleSubmitWord handles a submit_word message
func (c *Client) handleSubmitWord(payload interface{}) {
	payloadMap, ok := payload.(map[string]interface{})
	if !ok {
		c.sendError(ErrCodeInvalidMessage, "Invalid payload")
		return
	}

	word, ok := payloadMap["word"].(string)
	if !ok || word == "" {
		c.sendError(ErrCodeInvalidMessage, "Word is required")
		return
	}

	err := c.session.SubmitWord(c.playerID, word)
	if err != nil {
		switch err {
		case domain.ErrNotYourTurn:
			c.sendError(ErrCodeNotYourTurn, "It's not your turn")
		case domain.ErrAlreadySubmitted:
			c.sendError(ErrCodeInvalidAction, "You have already submitted")
		case domain.ErrInvalidPhase:
			c.sendError(ErrCodeInvalidAction, "Cannot submit now")
		default:
			c.sendError(ErrCodeInternalError, err.Error())
		}
		return
	}
}

// handleCastVote handles a cast_vote message
func (c *Client) handleCastVote(payload interface{}) {
	payloadMap, ok := payload.(map[string]interface{})
	if !ok {
		c.sendError(ErrCodeInvalidMessage, "Invalid payload")
		return
	}

	targetID, ok := payloadMap["targetPlayerId"].(string)
	if !ok || targetID == "" {
		c.sendError(ErrCodeInvalidMessage, "Target player ID is required")
		return
	}

	err := c.session.CastVote(c.playerID, targetID)
	if err != nil {
		switch err {
		case domain.ErrAlreadyVoted:
			c.sendError(ErrCodeAlreadyVoted, "You have already voted")
		case domain.ErrCannotVoteSelf:
			c.sendError(ErrCodeCannotVoteSelf, "Cannot vote for yourself")
		case domain.ErrInvalidPhase:
			c.sendError(ErrCodeInvalidAction, "Cannot vote now")
		default:
			c.sendError(ErrCodeInternalError, err.Error())
		}
		return
	}
}

// handleRequestNewRound handles a request_new_round message
func (c *Client) handleRequestNewRound() {
	err := c.session.StartNewRound(c.playerID)
	if err != nil {
		switch err {
		case domain.ErrNotHost:
			c.sendError(ErrCodeNotHost, "Only the host can start a new round")
		case domain.ErrInvalidPhase:
			c.sendError(ErrCodeInvalidAction, "Cannot start new round now")
		default:
			c.sendError(ErrCodeInternalError, err.Error())
		}
		return
	}
}

// sendConnected sends the connected message to the client
func (c *Client) sendConnected() {
	payload := &ConnectedPayload{
		PlayerID:  c.playerID,
		GameID:    c.session.GetRoomCode(),
		GameState: c.session.GetGameState(c.playerID),
	}

	msg := NewServerMessage(MsgConnected, payload)
	c.Send(msg)
}

// sendError sends an error message to the client
func (c *Client) sendError(code, message string) {
	payload := &ErrorPayload{
		Code:    code,
		Message: message,
	}

	msg := NewServerMessage(MsgError, payload)
	c.Send(msg)
}

// sendPong sends a pong message in response to ping
func (c *Client) sendPong() {
	msg := NewServerMessage(MsgPong, nil)
	c.Send(msg)
}

