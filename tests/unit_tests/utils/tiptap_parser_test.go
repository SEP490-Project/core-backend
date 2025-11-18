package utils

import (
	"core-backend/pkg/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTiptapJSON(t *testing.T) {
	// Read example JSON
	exampleJSON := `{
  "type": "doc",
  "content": [
    {
      "type": "paragraph",
      "attrs": { "textAlign": null },
      "content": [
        {
          "text": "In the world of beauty, true radiance starts with skin that feels alive, healthy, and deeply nourished. That's exactly what ",
          "type": "text"
        },
        {
          "text": "Dior's Capture Totale Super Potent Serum",
          "type": "text",
          "marks": [{ "type": "bold" }]
        },
        { "text": " promises — and delivers.", "type": "text" }
      ]
    },
    {
      "type": "image",
      "attrs": {
        "alt": null,
        "src": "https://bshowsell-public.s3.ap-southeast-1.amazonaws.com/test.jpg",
        "title": null,
        "width": null,
        "height": null
      }
    },
    {
      "type": "heading",
      "attrs": { "level": 3, "textAlign": null },
      "content": [
        { "text": "The Science of Youth: How It Works", "type": "text" }
      ]
    },
    {
      "type": "bulletList",
      "content": [
        {
          "type": "listItem",
          "content": [
            {
              "type": "paragraph",
              "attrs": { "textAlign": null },
              "content": [
                { "text": "Boosted firmness and elasticity", "type": "text" }
              ]
            }
          ]
        },
        {
          "type": "listItem",
          "content": [
            {
              "type": "paragraph",
              "attrs": { "textAlign": null },
              "content": [{ "text": "Deep, lasting hydration", "type": "text" }]
            }
          ]
        }
      ]
    },
    {
      "type": "blockquote",
      "content": [
        {
          "type": "paragraph",
          "attrs": { "textAlign": null },
          "content": [
            {
              "text": "My skin has never felt this alive",
              "type": "text"
            },
            {
              "text": "Dior customer review",
              "type": "text",
              "marks": [{ "type": "italic" }]
            }
          ]
        }
      ]
    }
  ]
}`

	result, err := utils.ParseTiptapJSON([]byte(exampleJSON))
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify plain text extraction
	assert.Contains(t, result.PlainText, "Dior's Capture Totale Super Potent Serum")
	assert.Contains(t, result.PlainText, "The Science of Youth: How It Works")
	assert.Contains(t, result.PlainText, "Boosted firmness and elasticity")
	assert.Contains(t, result.PlainText, "My skin has never felt this alive")

	// Verify image extraction
	assert.True(t, result.HasImages)
	assert.Equal(t, 1, len(result.ImageURLs))
	assert.Equal(t, "https://bshowsell-public.s3.ap-southeast-1.amazonaws.com/test.jpg", result.ImageURLs[0])

	// Verify list formatting
	assert.Contains(t, result.PlainText, "• Boosted firmness and elasticity")
	assert.Contains(t, result.PlainText, "• Deep, lasting hydration")

	// Verify blockquote formatting
	assert.Contains(t, result.PlainText, "> ")
}

func TestGetFirstImageURL(t *testing.T) {
	jsonWithImage := `{
  "type": "doc",
  "content": [
    {
      "type": "paragraph",
      "content": [{"text": "Test text", "type": "text"}]
    },
    {
      "type": "image",
      "attrs": {
        "src": "https://example.com/image1.jpg"
      }
    },
    {
      "type": "image",
      "attrs": {
        "src": "https://example.com/image2.jpg"
      }
    }
  ]
}`

	url, err := utils.GetFirstImageURL([]byte(jsonWithImage))
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/image1.jpg", url)

	// Test with no images
	jsonNoImage := `{
  "type": "doc",
  "content": [
    {
      "type": "paragraph",
      "content": [{"text": "Test text", "type": "text"}]
    }
  ]
}`

	url, err = utils.GetFirstImageURL([]byte(jsonNoImage))
	assert.NoError(t, err)
	assert.Equal(t, "", url)
}

func TestGetPlainTextPreview(t *testing.T) {
	jsonData := `{
  "type": "doc",
  "content": [
    {
      "type": "paragraph",
      "content": [
        {"text": "This is a very long text that should be truncated at some point to create a nice preview for the user to read.", "type": "text"}
      ]
    }
  ]
}`

	preview, err := utils.GetPlainTextPreview([]byte(jsonData), 50)
	assert.NoError(t, err)
	assert.LessOrEqual(t, len(preview), 54) // 50 + "..." + tolerance for word boundary
	assert.Contains(t, preview, "...")
	assert.Contains(t, preview, "This is a very long text")
}

func TestParseTiptapJSON_OrderedList(t *testing.T) {
	jsonData := `{
  "type": "doc",
  "content": [
    {
      "type": "orderedList",
      "attrs": { "start": 1 },
      "content": [
        {
          "type": "listItem",
          "content": [
            {
              "type": "paragraph",
              "content": [{"text": "First item", "type": "text"}]
            }
          ]
        },
        {
          "type": "listItem",
          "content": [
            {
              "type": "paragraph",
              "content": [{"text": "Second item", "type": "text"}]
            }
          ]
        }
      ]
    }
  ]
}`

	result, err := utils.ParseTiptapJSON([]byte(jsonData))
	assert.NoError(t, err)
	assert.Contains(t, result.PlainText, "1. First item")
	assert.Contains(t, result.PlainText, "2. Second item")
}

func TestParseTiptapJSON_HardBreak(t *testing.T) {
	jsonData := `{
  "type": "doc",
  "content": [
    {
      "type": "paragraph",
      "content": [
        {"text": "Line 1", "type": "text"},
        {"type": "hardBreak"},
        {"text": "Line 2", "type": "text"}
      ]
    }
  ]
}`

	result, err := utils.ParseTiptapJSON([]byte(jsonData))
	assert.NoError(t, err)
	assert.Contains(t, result.PlainText, "Line 1\nLine 2")
}
