// Structure of Server-Side-Events (SSE) Model in Popcorn.

package entity

// Data to be broadcasted to a client.
type SSEData struct {
	Data interface{} `json:"message"`
	// Type is the type of Data instance, for example gangInvite request can be a type or gangJoin
	Type string `json:"type"`
	To   string `json:"-"`
}

// Uniquely defines an incoming client.
type SSEClient struct {
	// Unique Client ID
	ID string
	// Client channel
	Channel chan SSEData
}

// Keeps track of every SSE events.
type SSE struct {
	// Data are pushed to this channel
	Message chan SSEData
	// New client connections
	NewClients chan SSEClient
	// Closed client connections
	ClosedClients chan SSEClient
	// Total client connections
	TotalClients map[string]chan SSEData
}
