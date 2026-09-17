package terminal

import (
	"strings"
	"sync"
)

type Buffer struct { mu sync.RWMutex; lines []string; maxLines int }
func NewBuffer(maxLines int) *Buffer { if maxLines < 100 { maxLines = 1000 }; return &Buffer{maxLines: maxLines} }
func (b *Buffer) Append(text string) { b.mu.Lock(); defer b.mu.Unlock(); for _, line := range strings.Split(strings.ReplaceAll(text, "\r", ""), "\n") { b.lines = append(b.lines, line) }; if len(b.lines) > b.maxLines { b.lines = append([]string(nil), b.lines[len(b.lines)-b.maxLines:]...) } }
func (b *Buffer) Lines() []string { b.mu.RLock(); defer b.mu.RUnlock(); return append([]string(nil), b.lines...) }
func (b *Buffer) Search(query string) []int { b.mu.RLock(); defer b.mu.RUnlock(); query = strings.ToLower(query); var out []int; if query == "" { return out }; for i, line := range b.lines { if strings.Contains(strings.ToLower(line), query) { out = append(out, i) } }; return out }
func (b *Buffer) Clear() { b.mu.Lock(); defer b.mu.Unlock(); b.lines = nil }

type Size struct { Width int; Height int }
func ClampSize(width, height int) Size { if width < 20 { width = 20 }; if height < 5 { height = 5 }; return Size{width, height} }
