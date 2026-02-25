package telegram

import (
	"strings"
	"testing"
)

func TestRenderTelegramHTMLFromMarkdownBasic(t *testing.T) {
	out, err := renderTelegramHTMLFromMarkdown("*hello* **world**")
	if err != nil {
		t.Fatalf("renderTelegramHTMLFromMarkdown() error = %v", err)
	}
	if !strings.Contains(out, "<em>hello</em>") {
		t.Fatalf("expected <em>hello</em>, got %q", out)
	}
	if !strings.Contains(out, "<strong>world</strong>") {
		t.Fatalf("expected <strong>world</strong>, got %q", out)
	}
}

func TestRenderTelegramHTMLFromMarkdownDisablesUnsupportedTags(t *testing.T) {
	input := "# title\n\n- a\n- b\n\n<script>alert(1)</script>"
	out, err := renderTelegramHTMLFromMarkdown(input)
	if err != nil {
		t.Fatalf("renderTelegramHTMLFromMarkdown() error = %v", err)
	}
	if strings.Contains(out, "<h1>") || strings.Contains(out, "<ul>") || strings.Contains(out, "<li>") || strings.Contains(out, "<script>") {
		t.Fatalf("unexpected unsupported tag in output: %q", out)
	}
	if !strings.Contains(out, "<b>title</b>") {
		t.Fatalf("expected heading to be converted to <b>, got %q", out)
	}
	if !strings.Contains(out, "* a") || !strings.Contains(out, "* b") {
		t.Fatalf("expected markdown list to be plain bullet lines, got %q", out)
	}
}

func TestRenderTelegramHTMLFromMarkdownKeepsAllowedSpecialTags(t *testing.T) {
	input := `<span class="tg-spoiler">secret</span>

<blockquote expandable>hidden quote</blockquote>

` + "```go\nfmt.Println(1)\n```"
	out, err := renderTelegramHTMLFromMarkdown(input)
	if err != nil {
		t.Fatalf("renderTelegramHTMLFromMarkdown() error = %v", err)
	}
	if !strings.Contains(out, `<span class="tg-spoiler">secret</span>`) {
		t.Fatalf("expected tg-spoiler span, got %q", out)
	}
	if !strings.Contains(out, `<blockquote expandable>hidden quote</blockquote>`) {
		t.Fatalf("expected expandable blockquote, got %q", out)
	}
	if !strings.Contains(out, `<pre><code class="language-go">`) {
		t.Fatalf("expected fenced code with language class, got %q", out)
	}
}

func TestRenderTelegramHTMLFromMarkdownNoExtraBlankLines(t *testing.T) {
	out, err := renderTelegramHTMLFromMarkdown("first paragraph\n\nsecond paragraph")
	if err != nil {
		t.Fatalf("renderTelegramHTMLFromMarkdown() error = %v", err)
	}
	if !strings.Contains(out, "\n\n") {
		t.Fatalf("expected paragraph blank line in output: %q", out)
	}
	if strings.Contains(out, "\n\n\n") {
		t.Fatalf("unexpected extra blank lines in output: %q", out)
	}
}

func TestRenderTelegramHTMLFromMarkdownKeepsInlineSpacing(t *testing.T) {
	out, err := renderTelegramHTMLFromMarkdown(`<b>hello</b> <i>world</i>`)
	if err != nil {
		t.Fatalf("renderTelegramHTMLFromMarkdown() error = %v", err)
	}
	if !strings.Contains(out, "</b> <i>") {
		t.Fatalf("expected inline space between tags, got %q", out)
	}
}

func TestRenderTelegramHTMLFromMarkdownPreAddsParagraphBreakAfterBlock(t *testing.T) {
	out, err := renderTelegramHTMLFromMarkdown("before\n\n```go\nfmt.Println(1)\n```\n\nafter")
	if err != nil {
		t.Fatalf("renderTelegramHTMLFromMarkdown() error = %v", err)
	}
	if !strings.Contains(out, "</pre>\n\nafter") {
		t.Fatalf("expected paragraph break after pre block, got %q", out)
	}
}

func TestRenderTelegramHTMLFromMarkdownBlockquoteAddsParagraphBreakAfterBlock(t *testing.T) {
	out, err := renderTelegramHTMLFromMarkdown("before\n\n> quote\n\nafter")
	if err != nil {
		t.Fatalf("renderTelegramHTMLFromMarkdown() error = %v", err)
	}
	if !strings.Contains(out, "</blockquote>\n\nafter") {
		t.Fatalf("expected paragraph break after blockquote block, got %q", out)
	}
}
