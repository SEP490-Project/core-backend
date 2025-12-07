package tiptap

// TiptapDocument represents the root of a Tiptap JSON document
type TiptapDocument struct {
	Type    string       `json:"type"`
	Content []TiptapNode `json:"content,omitempty"`
}

// TiptapNode represents a node in the Tiptap document tree
type TiptapNode struct {
	Type    string         `json:"type"`
	Attrs   map[string]any `json:"attrs,omitempty"`
	Content []TiptapNode   `json:"content,omitempty"`
	Text    string         `json:"text,omitempty"`
	Marks   []TiptapMark   `json:"marks,omitempty"`
}

// TiptapMark represents inline formatting (bold, italic, etc.)
type TiptapMark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs,omitempty"`
}
