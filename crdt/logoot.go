package crdt

// Document represents a document that would be edited by the users.
type Document struct {
	siteID uint8
	pairs  []pair
}

// pair is represents a smaller unit of a document.
type pair struct {
	pos  []Position
	atom string
}

// Position represents a position in the document.
type Position struct {
	Identifier uint16
	SiteID     uint8
}
