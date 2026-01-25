package tiptap

import "encoding/json"

// Builder helps construct and modify Tiptap documents
type Builder struct {
	doc *TiptapDocument
}

// NewBuilder creates a new empty Tiptap document builder
func NewBuilder() *Builder {
	return &Builder{
		doc: &TiptapDocument{
			Type:    "doc",
			Content: []TiptapNode{},
		},
	}
}

// FromJSON creates a builder from existing Tiptap JSON
func FromJSON(jsonData []byte) (*Builder, error) {
	var doc TiptapDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		return nil, err
	}
	return &Builder{doc: &doc}, nil
}

// AddParagraphText adds a simple text paragraph to the end of the document
func (b *Builder) AddParagraphText(text string) *Builder {
	node := TiptapNode{
		Type: "paragraph",
		Content: []TiptapNode{
			{
				Type: "text",
				Text: text,
			},
		},
	}
	b.doc.Content = append(b.doc.Content, node)
	return b
}

// AddLinkParagraph adds a paragraph containing a link
func (b *Builder) AddLinkParagraph(label, text, url string) *Builder {
	paragraphContent := make([]TiptapNode, 0)
	if label != "" {
		paragraphContent = append(paragraphContent, TiptapNode{
			Type: "text",
			Text: label,
		})
	}
	paragraphContent = append(paragraphContent, TiptapNode{
		Type: "text",
		Text: text,
		Marks: []TiptapMark{
			{
				Type: "link",
				Attrs: map[string]any{
					"href":   url,
					"target": "_blank",
				},
			},
		}})

	node := TiptapNode{
		Type:    "paragraph",
		Content: paragraphContent,
	}
	b.doc.Content = append(b.doc.Content, node)
	return b
}

// AppendNode appends a node to the document content
func (b *Builder) AppendNode(node TiptapNode) *Builder {
	b.doc.Content = append(b.doc.Content, node)
	return b
}

// Build returns the JSON representation of the document
func (b *Builder) Build() ([]byte, error) {
	return json.Marshal(b.doc)
}

// BuildObject returns the underlying TiptapDocument object
func (b *Builder) BuildObject() *TiptapDocument {
	return b.doc
}

// ReplaceLinkURL replaces all occurrences of trackingURL with affiliateURL in link marks
// This is used to render channel-specific affiliate links at read time
func (b *Builder) ReplaceLinkURL(trackingURL, affiliateURL string) *Builder {
	for i := range b.doc.Content {
		replaceLinkInNode(&b.doc.Content[i], trackingURL, affiliateURL)
	}
	return b
}

// replaceLinkInNode recursively replaces link URLs in a node and its children
func replaceLinkInNode(node *TiptapNode, trackingURL, affiliateURL string) {
	// Check marks for link type
	for i := range node.Marks {
		if node.Marks[i].Type == "link" {
			if href, ok := node.Marks[i].Attrs["href"].(string); ok && href == trackingURL {
				node.Marks[i].Attrs["href"] = affiliateURL
			}
		}
	}

	// Recursively process children
	for i := range node.Content {
		replaceLinkInNode(&node.Content[i], trackingURL, affiliateURL)
	}
}

// RenderWithAffiliateLink creates a new body with tracking URL replaced by affiliate URL
func RenderWithAffiliateLink(body []byte, trackingURL, affiliateURL string) ([]byte, error) {
	builder, err := FromJSON(body)
	if err != nil {
		return nil, err
	}
	return builder.ReplaceLinkURL(trackingURL, affiliateURL).Build()
}
