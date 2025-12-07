package tiptap

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TiptapParseResult contains extracted content from Tiptap JSON
type TiptapParseResult struct {
	PlainText string   // Plain text content suitable for social media posts
	ImageURLs []string // All image URLs found in the document
	HasImages bool     // Quick check if document contains images
}

// ParseTiptapJSON parses Tiptap editor JSON and extracts text and media
func ParseTiptapJSON(jsonData []byte) (*TiptapParseResult, error) {
	var doc TiptapDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse Tiptap JSON: %w", err)
	}

	result := &TiptapParseResult{
		ImageURLs: make([]string, 0),
	}

	var textBuilder strings.Builder
	parseTiptapNode(&doc, &textBuilder, result, 0)

	result.PlainText = strings.TrimSpace(textBuilder.String())
	result.HasImages = len(result.ImageURLs) > 0

	return result, nil
}

// parseTiptapNode recursively parses Tiptap nodes
func parseTiptapNode(node any, builder *strings.Builder, result *TiptapParseResult, depth int) {
	switch n := node.(type) {
	case *TiptapDocument:
		for _, child := range n.Content {
			parseTiptapNode(&child, builder, result, depth)
		}
	case *TiptapNode:
		switch n.Type {
		case "paragraph":
			// Paragraph: extract text content
			for _, child := range n.Content {
				parseTiptapNode(&child, builder, result, depth)
			}
			builder.WriteString("\n\n")

		case "heading":
			// Heading: extract text with emphasis
			for _, child := range n.Content {
				parseTiptapNode(&child, builder, result, depth)
			}
			builder.WriteString("\n\n")

		case "bulletList":
			// Bullet list: each item on new line with bullet
			for _, item := range n.Content {
				if item.Type == "listItem" {
					builder.WriteString("• ")
					for _, child := range item.Content {
						parseTiptapNode(&child, builder, result, depth+1)
					}
				}
			}
			builder.WriteString("\n")

		case "orderedList":
			// Ordered list: each item with number
			for i, item := range n.Content {
				if item.Type == "listItem" {
					fmt.Fprintf(builder, "%d. ", i+1)
					for _, child := range item.Content {
						parseTiptapNode(&child, builder, result, depth+1)
					}
				}
			}
			builder.WriteString("\n")

		case "listItem":
			// List item content (for nested parsing)
			for _, child := range n.Content {
				parseTiptapNode(&child, builder, result, depth)
			}
			// Only add newline if not already in a list context
			if depth == 0 {
				builder.WriteString("\n")
			}

		case "blockquote":
			// Blockquote: add quotation marks or prefix
			builder.WriteString("> ")
			for _, child := range n.Content {
				parseTiptapNode(&child, builder, result, depth+1)
			}
			builder.WriteString("\n\n")

		case "image":
			// Image: extract URL and don't add to text
			if n.Attrs != nil {
				if src, ok := n.Attrs["src"].(string); ok && src != "" {
					result.ImageURLs = append(result.ImageURLs, src)
				}
			}
			// Images are handled separately, don't add to text

		case "hardBreak":
			// Hard break: newline
			builder.WriteString("\n")

		case "text":
			// Text node: extract plain text (marks like bold/italic ignored for social media)
			if n.Text != "" {
				prefix := ""
				suffix := ""
				linkSuffix := ""

				// Iterate over marks to determine wrapping and links
				for _, mark := range n.Marks {
					switch mark.Type {
					case "bold":
						// Optional: Wrap with * for bold
						prefix += "*"
						suffix = "*" + suffix

					case "italic":
						// Optional: Wrap with _ for italic
						prefix += "_"
						suffix = "_" + suffix

					case "strike":
						// Wrap with ~ to indicate crossed out text
						prefix += "~"
						suffix = "~" + suffix

					case "code":
						// Wrap with backticks for code
						prefix += "`"
						suffix = "`" + suffix

					case "link":
						// Extract href to append AT THE END
						if href, ok := mark.Attrs["href"].(string); ok && href != "" {
							linkSuffix = fmt.Sprintf(" (%s)", href)
						}
					}
				}

				// 1. Write the Prefix (e.g., " *` ")
				builder.WriteString(prefix)

				// 2. Write the actual Text
				builder.WriteString(n.Text)

				// 3. Write the Suffix (mirror of prefix, e.g., " `* ")
				builder.WriteString(suffix)

				// 4. Append the Link URL (outside the formatting)
				builder.WriteString(linkSuffix)
			}

		default:
			// Unknown node type: try to extract children
			for _, child := range n.Content {
				parseTiptapNode(&child, builder, result, depth)
			}
		}
	}
}

// GetFirstImageURL returns the first image URL from Tiptap JSON, or empty string if none
func GetFirstImageURL(jsonData []byte) (string, error) {
	result, err := ParseTiptapJSON(jsonData)
	if err != nil {
		return "", err
	}

	if len(result.ImageURLs) > 0 {
		return result.ImageURLs[0], nil
	}

	return "", nil
}

// GetPlainTextPreview returns first N characters of plain text for previews
func GetPlainTextPreview(jsonData []byte, maxLength int) (string, error) {
	result, err := ParseTiptapJSON(jsonData)
	if err != nil {
		return "", err
	}

	if len(result.PlainText) <= maxLength {
		return result.PlainText, nil
	}

	// Truncate at word boundary
	truncated := result.PlainText[:maxLength]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "...", nil
}
