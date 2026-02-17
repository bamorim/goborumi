package script

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	docType         = "doc"
	blockHeading    = "heading"
	blockParagraph  = "paragraph"
	blockBulletList = "bullet_list"
	blockOrderList  = "ordered_list"
	blockQuote      = "quote"
	blockCode       = "code"
)

const (
	borumiTypeHeading     = "heading"
	borumiTypeParagraph   = "paragraph"
	borumiTypeBulletList  = "bullet-list"
	borumiTypeOrderedList = "ordered-list"
	borumiTypeQuote       = "blockquote"
	borumiTypeCode        = "code-block"
	borumiTypeText        = "text"
)

var (
	headingPattern = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	bulletPattern  = regexp.MustCompile(`^\s*-\s+(.+)$`)
	orderedPattern = regexp.MustCompile(`^\s*(\d+)\.\s+(.+)$`)
	quotePattern   = regexp.MustCompile(`^\s*>\s?(.*)$`)
)

type Document struct {
	Type   string  `json:"type"`
	Blocks []Block `json:"blocks"`
}

type Block struct {
	Type     string   `json:"type"`
	Text     string   `json:"text,omitempty"`
	Level    int      `json:"level,omitempty"`
	Items    []string `json:"items,omitempty"`
	Language string   `json:"language,omitempty"`
}

type borumiDocument struct {
	Selection any          `json:"selection"`
	Content   []borumiNode `json:"content"`
}

type borumiNode struct {
	Type     string       `json:"type"`
	Content  []borumiNode `json:"content,omitempty"`
	Text     string       `json:"text,omitempty"`
	Level    int          `json:"level,omitempty"`
	Language string       `json:"language,omitempty"`
	Bold     bool         `json:"bold,omitempty"`
	Italic   bool         `json:"italic,omitempty"`
	Code     bool         `json:"code,omitempty"`
	Strike   bool         `json:"strike,omitempty"`
}

func JSONToMarkdown(raw string) (string, error) {
	doc, err := ParseJSON(raw)
	if err != nil {
		return "", err
	}
	return RenderMarkdown(doc), nil
}

func MarkdownToJSON(markdown string) (string, error) {
	doc, err := ParseMarkdown(markdown)
	if err != nil {
		return "", err
	}
	return renderBorumiJSON(doc)
}

// NormalizeOrConvertToJSON validates JSON input or converts markdown input
// into the script JSON representation used by Borumi.
func NormalizeOrConvertToJSON(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return renderBorumiJSON(Document{Type: docType, Blocks: []Block{}})
	}

	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		if _, err := ParseJSON(trimmed); err != nil {
			return "", fmt.Errorf("invalid script json: %w", err)
		}

		var value any
		if err := json.Unmarshal([]byte(trimmed), &value); err != nil {
			return "", fmt.Errorf("invalid script json: %w", err)
		}

		raw, err := json.Marshal(value)
		if err != nil {
			return "", fmt.Errorf("could not normalize script json: %w", err)
		}
		return string(raw), nil
	}

	return MarkdownToJSON(input)
}

func ParseJSON(raw string) (Document, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Document{}, fmt.Errorf("empty script")
	}

	if strings.HasPrefix(trimmed, "{") {
		var object map[string]json.RawMessage
		if err := json.Unmarshal([]byte(trimmed), &object); err != nil {
			return Document{}, fmt.Errorf("invalid json object: %w", err)
		}

		if _, ok := object["content"]; ok {
			var borumiDoc borumiDocument
			if err := json.Unmarshal([]byte(trimmed), &borumiDoc); err != nil {
				return Document{}, fmt.Errorf("invalid borumi script json: %w", err)
			}
			return borumiToDocument(borumiDoc), nil
		}

		if _, hasType := object["type"]; hasType {
			return parseInternalDocumentObject(trimmed)
		}
		if _, hasBlocks := object["blocks"]; hasBlocks {
			return parseInternalDocumentObject(trimmed)
		}

		return Document{}, fmt.Errorf("unsupported script json shape")
	}

	if strings.HasPrefix(trimmed, "[") {
		var blocks []Block
		if err := json.Unmarshal([]byte(trimmed), &blocks); err != nil {
			return Document{}, fmt.Errorf("invalid json block array: %w", err)
		}

		doc := Document{
			Type:   docType,
			Blocks: blocks,
		}
		doc.normalize()
		if err := doc.validate(); err != nil {
			return Document{}, err
		}
		return doc, nil
	}

	return Document{}, fmt.Errorf("unsupported script json shape")
}

func parseInternalDocumentObject(raw string) (Document, error) {
	var doc Document
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return Document{}, fmt.Errorf("invalid json document: %w", err)
	}
	doc.normalize()
	if err := doc.validate(); err != nil {
		return Document{}, err
	}
	return doc, nil
}

func ParseMarkdown(markdown string) (Document, error) {
	lines := splitLines(markdown)
	doc := Document{
		Type:   docType,
		Blocks: make([]Block, 0),
	}

	for i := 0; i < len(lines); {
		currentLine := lines[i]
		trimmedLine := strings.TrimSpace(currentLine)
		if trimmedLine == "" {
			i++
			continue
		}

		if strings.HasPrefix(trimmedLine, "```") {
			lang := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "```"))
			i++
			codeLines := make([]string, 0)
			for i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
				codeLines = append(codeLines, lines[i])
				i++
			}
			if i >= len(lines) {
				return Document{}, fmt.Errorf("unclosed code fence")
			}
			i++
			doc.Blocks = append(doc.Blocks, Block{
				Type:     blockCode,
				Text:     strings.Join(codeLines, "\n"),
				Language: lang,
			})
			continue
		}

		if match := headingPattern.FindStringSubmatch(trimmedLine); len(match) == 3 {
			doc.Blocks = append(doc.Blocks, Block{
				Type:  blockHeading,
				Level: len(match[1]),
				Text:  strings.TrimSpace(match[2]),
			})
			i++
			continue
		}

		if match := bulletPattern.FindStringSubmatch(lines[i]); len(match) == 2 {
			items := make([]string, 0)
			for i < len(lines) {
				next := bulletPattern.FindStringSubmatch(lines[i])
				if len(next) != 2 {
					break
				}
				items = append(items, strings.TrimSpace(next[1]))
				i++
			}
			doc.Blocks = append(doc.Blocks, Block{
				Type:  blockBulletList,
				Items: items,
			})
			continue
		}

		if match := orderedPattern.FindStringSubmatch(lines[i]); len(match) == 3 {
			items := make([]string, 0)
			for i < len(lines) {
				next := orderedPattern.FindStringSubmatch(lines[i])
				if len(next) != 3 {
					break
				}
				items = append(items, strings.TrimSpace(next[2]))
				i++
			}
			doc.Blocks = append(doc.Blocks, Block{
				Type:  blockOrderList,
				Items: items,
			})
			continue
		}

		if match := quotePattern.FindStringSubmatch(lines[i]); len(match) == 2 {
			parts := make([]string, 0)
			for i < len(lines) {
				next := quotePattern.FindStringSubmatch(lines[i])
				if len(next) != 2 {
					break
				}
				parts = append(parts, next[1])
				i++
			}
			doc.Blocks = append(doc.Blocks, Block{
				Type: blockQuote,
				Text: strings.Join(parts, "\n"),
			})
			continue
		}

		parts := make([]string, 0)
		for i < len(lines) {
			line := lines[i]
			if strings.TrimSpace(line) == "" || isBlockStart(line) {
				break
			}
			parts = append(parts, line)
			i++
		}
		doc.Blocks = append(doc.Blocks, Block{
			Type: blockParagraph,
			Text: strings.Join(parts, "\n"),
		})
		if len(parts) == 0 {
			i++
		}
	}

	return doc, nil
}

// ToJSON serializes the internal normalized document representation.
func (d Document) ToJSON() (string, error) {
	d.normalize()
	if err := d.validate(); err != nil {
		return "", err
	}

	raw, err := json.Marshal(d)
	if err != nil {
		return "", fmt.Errorf("could not serialize script json: %w", err)
	}
	return string(raw), nil
}

func RenderMarkdown(doc Document) string {
	doc.normalize()
	if len(doc.Blocks) == 0 {
		return ""
	}

	parts := make([]string, 0, len(doc.Blocks))
	for _, block := range doc.Blocks {
		switch block.Type {
		case blockHeading:
			level := block.Level
			if level < 1 {
				level = 1
			}
			if level > 6 {
				level = 6
			}
			parts = append(parts, strings.Repeat("#", level)+" "+block.Text)
		case blockParagraph:
			parts = append(parts, block.Text)
		case blockBulletList:
			items := make([]string, 0, len(block.Items))
			for _, item := range block.Items {
				items = append(items, "- "+item)
			}
			parts = append(parts, strings.Join(items, "\n"))
		case blockOrderList:
			items := make([]string, 0, len(block.Items))
			for i, item := range block.Items {
				items = append(items, strconv.Itoa(i+1)+". "+item)
			}
			parts = append(parts, strings.Join(items, "\n"))
		case blockQuote:
			lines := splitLines(block.Text)
			quoted := make([]string, 0, len(lines))
			for _, line := range lines {
				if line == "" {
					quoted = append(quoted, ">")
					continue
				}
				quoted = append(quoted, "> "+line)
			}
			parts = append(parts, strings.Join(quoted, "\n"))
		case blockCode:
			var b strings.Builder
			b.WriteString("```")
			b.WriteString(block.Language)
			b.WriteString("\n")
			b.WriteString(block.Text)
			b.WriteString("\n```")
			parts = append(parts, b.String())
		default:
			parts = append(parts, block.Text)
		}
	}

	return strings.Join(parts, "\n\n")
}

func renderBorumiJSON(doc Document) (string, error) {
	doc.normalize()
	if err := doc.validate(); err != nil {
		return "", err
	}

	raw, err := json.Marshal(documentToBorumi(doc))
	if err != nil {
		return "", fmt.Errorf("could not serialize borumi script json: %w", err)
	}
	return string(raw), nil
}

func documentToBorumi(doc Document) borumiDocument {
	content := make([]borumiNode, 0, len(doc.Blocks))
	for _, block := range doc.Blocks {
		switch block.Type {
		case blockHeading:
			level := block.Level
			if level < 1 {
				level = 1
			}
			if level > 6 {
				level = 6
			}
			content = append(content, borumiNode{
				Type:    borumiTypeHeading,
				Level:   level,
				Content: borumiTextContent(block.Text),
			})
		case blockParagraph:
			content = append(content, borumiNode{
				Type:    borumiTypeParagraph,
				Content: borumiTextContent(block.Text),
			})
		case blockBulletList:
			items := make([]borumiNode, 0, len(block.Items))
			for _, item := range block.Items {
				items = append(items, borumiNode{
					Content: []borumiNode{
						{
							Type:    borumiTypeParagraph,
							Content: borumiTextContent(item),
						},
					},
				})
			}
			content = append(content, borumiNode{
				Type:    borumiTypeBulletList,
				Content: items,
			})
		case blockOrderList:
			items := make([]borumiNode, 0, len(block.Items))
			for _, item := range block.Items {
				items = append(items, borumiNode{
					Content: []borumiNode{
						{
							Type:    borumiTypeParagraph,
							Content: borumiTextContent(item),
						},
					},
				})
			}
			content = append(content, borumiNode{
				Type:    borumiTypeOrderedList,
				Content: items,
			})
		case blockQuote:
			lines := splitLines(block.Text)
			quoteBlocks := make([]borumiNode, 0, len(lines))
			for _, line := range lines {
				quoteBlocks = append(quoteBlocks, borumiNode{
					Type:    borumiTypeParagraph,
					Content: borumiTextContent(line),
				})
			}
			content = append(content, borumiNode{
				Type:    borumiTypeQuote,
				Content: quoteBlocks,
			})
		case blockCode:
			content = append(content, borumiNode{
				Type:     borumiTypeCode,
				Language: block.Language,
				Text:     block.Text,
			})
		default:
			content = append(content, borumiNode{
				Type:    borumiTypeParagraph,
				Content: borumiTextContent(block.Text),
			})
		}
	}

	return borumiDocument{
		Selection: nil,
		Content:   content,
	}
}

func borumiToDocument(doc borumiDocument) Document {
	out := Document{
		Type:   docType,
		Blocks: make([]Block, 0),
	}

	for _, node := range doc.Content {
		out.Blocks = append(out.Blocks, borumiNodeToBlocks(node)...)
	}

	return out
}

func borumiNodeToBlocks(node borumiNode) []Block {
	switch node.Type {
	case borumiTypeHeading:
		level := node.Level
		if level < 1 {
			level = 1
		}
		if level > 6 {
			level = 6
		}
		return []Block{{
			Type:  blockHeading,
			Level: level,
			Text:  inlineFromBorumi(node),
		}}
	case borumiTypeParagraph:
		return []Block{{
			Type: blockParagraph,
			Text: inlineFromBorumi(node),
		}}
	case borumiTypeBulletList:
		items := make([]string, 0, len(node.Content))
		for _, item := range node.Content {
			text := strings.TrimSpace(listItemFromBorumi(item))
			if text != "" {
				items = append(items, text)
			}
		}
		if len(items) == 0 {
			return nil
		}
		return []Block{{
			Type:  blockBulletList,
			Items: items,
		}}
	case borumiTypeOrderedList:
		items := make([]string, 0, len(node.Content))
		for _, item := range node.Content {
			text := strings.TrimSpace(listItemFromBorumi(item))
			if text != "" {
				items = append(items, text)
			}
		}
		if len(items) == 0 {
			return nil
		}
		return []Block{{
			Type:  blockOrderList,
			Items: items,
		}}
	case borumiTypeQuote, blockQuote:
		lines := make([]string, 0, len(node.Content))
		for _, child := range node.Content {
			text := strings.TrimSpace(blockTextFromBorumi(child))
			if text != "" {
				lines = append(lines, text)
			}
		}
		return []Block{{
			Type: blockQuote,
			Text: strings.Join(lines, "\n"),
		}}
	case borumiTypeCode, blockCode:
		text := node.Text
		if text == "" {
			text = inlineFromBorumi(node)
		}
		return []Block{{
			Type:     blockCode,
			Text:     text,
			Language: node.Language,
		}}
	case borumiTypeText:
		return []Block{{
			Type: blockParagraph,
			Text: inlineTextFromBorumiNode(node),
		}}
	default:
		if len(node.Content) > 0 {
			text := blockTextFromBorumi(node)
			if text == "" {
				return nil
			}
			return []Block{{
				Type: blockParagraph,
				Text: text,
			}}
		}
		if strings.TrimSpace(node.Text) != "" {
			return []Block{{
				Type: blockParagraph,
				Text: node.Text,
			}}
		}
		return nil
	}
}

func inlineFromBorumi(node borumiNode) string {
	if len(node.Content) == 0 {
		if node.Type == borumiTypeText {
			return inlineTextFromBorumiNode(node)
		}
		return node.Text
	}

	var b strings.Builder
	for _, child := range node.Content {
		b.WriteString(inlineFromBorumi(child))
	}
	return b.String()
}

func listItemFromBorumi(node borumiNode) string {
	if len(node.Content) == 0 {
		return inlineFromBorumi(node)
	}

	parts := make([]string, 0, len(node.Content))
	for _, child := range node.Content {
		text := strings.TrimSpace(blockTextFromBorumi(child))
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

func blockTextFromBorumi(node borumiNode) string {
	switch node.Type {
	case borumiTypeParagraph, borumiTypeHeading, borumiTypeText:
		return inlineFromBorumi(node)
	case borumiTypeBulletList:
		items := make([]string, 0, len(node.Content))
		for _, child := range node.Content {
			text := strings.TrimSpace(listItemFromBorumi(child))
			if text != "" {
				items = append(items, "- "+text)
			}
		}
		return strings.Join(items, "\n")
	case borumiTypeOrderedList:
		items := make([]string, 0, len(node.Content))
		for i, child := range node.Content {
			text := strings.TrimSpace(listItemFromBorumi(child))
			if text != "" {
				items = append(items, strconv.Itoa(i+1)+". "+text)
			}
		}
		return strings.Join(items, "\n")
	case borumiTypeQuote, blockQuote:
		lines := make([]string, 0, len(node.Content))
		for _, child := range node.Content {
			text := strings.TrimSpace(blockTextFromBorumi(child))
			if text != "" {
				lines = append(lines, text)
			}
		}
		return strings.Join(lines, "\n")
	case borumiTypeCode, blockCode:
		if node.Text != "" {
			return node.Text
		}
	}

	if len(node.Content) > 0 {
		parts := make([]string, 0, len(node.Content))
		for _, child := range node.Content {
			text := strings.TrimSpace(blockTextFromBorumi(child))
			if text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	}

	return node.Text
}

func inlineTextFromBorumiNode(node borumiNode) string {
	text := node.Text
	if node.Code {
		text = "`" + text + "`"
	}
	if node.Bold {
		text = "**" + text + "**"
	}
	if node.Italic {
		text = "*" + text + "*"
	}
	if node.Strike {
		text = "~~" + text + "~~"
	}
	return text
}

func borumiTextContent(text string) []borumiNode {
	return []borumiNode{
		{
			Type: borumiTypeText,
			Text: text,
		},
	}
}

func (d *Document) normalize() {
	if d.Type == "" {
		d.Type = docType
	}
	if d.Blocks == nil {
		d.Blocks = []Block{}
	}
}

func (d Document) validate() error {
	if d.Type != docType {
		return fmt.Errorf("invalid document type %q", d.Type)
	}

	for i, block := range d.Blocks {
		switch block.Type {
		case blockHeading:
			if block.Level < 1 || block.Level > 6 {
				return fmt.Errorf("block %d: heading level must be 1..6", i)
			}
		case blockParagraph:
		case blockBulletList:
			if len(block.Items) == 0 {
				return fmt.Errorf("block %d: bullet_list requires at least one item", i)
			}
		case blockOrderList:
			if len(block.Items) == 0 {
				return fmt.Errorf("block %d: ordered_list requires at least one item", i)
			}
		case blockQuote:
		case blockCode:
		default:
			return fmt.Errorf("block %d: unsupported type %q", i, block.Type)
		}
	}
	return nil
}

func splitLines(input string) []string {
	input = strings.ReplaceAll(input, "\r\n", "\n")
	input = strings.ReplaceAll(input, "\r", "\n")
	return strings.Split(input, "\n")
}

func isBlockStart(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	if strings.HasPrefix(trimmed, "```") {
		return true
	}
	if headingPattern.MatchString(trimmed) {
		return true
	}
	if bulletPattern.MatchString(trimmed) {
		return true
	}
	if orderedPattern.MatchString(trimmed) {
		return true
	}
	if quotePattern.MatchString(trimmed) {
		return true
	}
	return false
}
