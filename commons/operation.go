package commons

// Operation represents a CRDT operation.
type Operation struct {
	// Type represents the operation type, for example, insert, delete.
	Type string `json:"type"`

	// Position represents the position at which the operation has been made.
	Position int `json:"position"`

	// Value represents the content of the operation. Mostly a character.
	Value string `json:"value"`
}
