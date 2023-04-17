package commons

import (
	"github.com/burntcarrot/pairpad/crdt"
	"github.com/google/uuid"
)

// Message represents the message sent over the wire.
type Message struct {
	Username string `json:"username"`

	// Text represents the body of the message. This is currently used for joining messages, the siteID, and the list of active users.
	Text string `json:"text"`

	// Type represents the message type.
	Type MessageType `json:"type"`

	// ID represents the client's UUID.
	ID uuid.UUID `json:"ID"`

	// Operation represents the CRDT operation.
	Operation Operation `json:"operation"`

	// Document represents the client's document. This is not used frequently, and should be only used when necessary, due to the large size of documents.
	Document crdt.Document `json:"document"`
}

// MessageType represents the type of the message.
type MessageType string

// Currently, pairpad supports 5 message types:
// - docSync (for syncing documents)
// - docReq (for requesting documents)
// - SiteID (for generating site IDs)
// - join (for joining messages)
// - users (for the list of active users)

const (
	DocSyncMessage MessageType = "docSync"
	DocReqMessage  MessageType = "docReq"
	SiteIDMessage  MessageType = "SiteID"
	JoinMessage    MessageType = "join"
	UsersMessage   MessageType = "users"
)
