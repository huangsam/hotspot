// Package provider implements the FormatProvider implementation for SVG heatmap output.
package provider

import (
	"fmt"
	"io"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/schema"
)

// HeatmapProvider implements the FormatProvider interface for SVG heatmap visualization.
type HeatmapProvider struct{}

// NewHeatmapProvider creates a new heatmap provider.
func NewHeatmapProvider() *HeatmapProvider {
	return &HeatmapProvider{}
}

// WriteFiles writes file analysis results as an SVG heatmap.
func (p *HeatmapProvider) WriteFiles(w io.Writer, files []schema.FileResult, output config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	return p.generateHeatmapSVG(w, files, output)
}

// WriteFolders writes folder analysis results as an SVG heatmap.
func (p *HeatmapProvider) WriteFolders(w io.Writer, folders []schema.FolderResult, output config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	files := make([]schema.FileResult, len(folders))
	for i, f := range folders {
		files[i] = schema.FileResult{
			Path:      f.Path,
			Mode:      f.Mode,
			ModeScore: f.Score,
			SizeBytes: int64(f.TotalLOC),
			Churn:     f.Churn,
		}
	}
	return p.generateHeatmapSVG(w, files, output)
}

// GenerateSVG creates an SVG heatmap visualization as a string.
func (p *HeatmapProvider) GenerateSVG(files []schema.FileResult, output config.OutputSettings) (string, error) {
	var b strings.Builder
	if err := p.generateHeatmapSVG(&b, files, output); err != nil {
		return "", err
	}
	return b.String(), nil
}

// WriteComparison is not implemented for heatmap.
func (p *HeatmapProvider) WriteComparison(_ io.Writer, _ schema.ComparisonResult, _ config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	return fmt.Errorf("heatmap output not supported for comparison results")
}

// WriteTimeseries is not implemented for heatmap.
func (p *HeatmapProvider) WriteTimeseries(_ io.Writer, _ schema.TimeseriesResult, _ config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	return fmt.Errorf("heatmap output not supported for timeseries results")
}

// WriteBlastRadius is not implemented for heatmap.
func (p *HeatmapProvider) WriteBlastRadius(_ io.Writer, _ schema.BlastRadiusResult, _ config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	return fmt.Errorf("heatmap output not supported for blast radius results")
}

// WriteMetrics is not implemented for heatmap.
func (p *HeatmapProvider) WriteMetrics(_ io.Writer, _ map[schema.ScoringMode]map[schema.BreakdownKey]float64, _ config.OutputSettings) error {
	return fmt.Errorf("heatmap output not supported for metrics")
}

// WriteHistory is not implemented for heatmap.
func (p *HeatmapProvider) WriteHistory(_ io.Writer, _ []schema.AnalysisRunRecord, _ config.OutputSettings) error {
	return fmt.Errorf("heatmap output not supported for history")
}

// ─── Layout types ────────────────────────────────────────────────────────────

// tmRect is an axis-aligned rectangle used during treemap layout.
type tmRect struct{ x, y, w, h float64 }

// tmItem is a single item to be laid out in the treemap.
type tmItem struct {
	label    string  // display name
	fullPath string  // full path (used in tooltip)
	score    float64 // hotspot score
	weight   float64 // area weight (score²; never zero)
	// filled during layout:
	rect tmRect
}

// ─── Entry point ─────────────────────────────────────────────────────────────

// generateHeatmapSVG creates an SVG treemap visualization (WinDirStat-style).
func (p *HeatmapProvider) generateHeatmapSVG(w io.Writer, files []schema.FileResult, _ config.OutputSettings) error {
	if len(files) == 0 {
		return fmt.Errorf("no files to visualize")
	}

	const (
		svgW   = 1200
		svgH   = 800
		padTop = 10  // minimal padding
		pad    = 4   // outer canvas padding
		gap    = 2   // gap between cells
		hdrH   = 18  // directory header band height
		minW   = 6.0 // minimum cell dimension to render label
		minH   = 6.0
	)

	// ── collect score range for normalisation ────────────────────────────────
	maxScore := 0.0
	for _, f := range files {
		if f.ModeScore > maxScore {
			maxScore = f.ModeScore
		}
	}
	if maxScore == 0 {
		maxScore = 1
	}

	dirMap, dirOrder := p.groupFilesByDirectory(files)

	// sort groups by total weight descending (largest area first)
	sort.Slice(dirOrder, func(i, j int) bool {
		return dirMap[dirOrder[i]].total > dirMap[dirOrder[j]].total
	})

	// ── treemap canvas (inside the SVG, below header) ────────────────────────
	canvasX := float64(pad)
	canvasY := float64(padTop)
	canvasW := float64(svgW - 2*pad)
	canvasH := float64(svgH - padTop - pad)

	// Build a flat list of groups-as-items for the outer layout pass.
	outerItems := make([]*tmItem, 0, len(dirOrder))
	for _, dir := range dirOrder {
		g := dirMap[dir]
		outerItems = append(outerItems, &tmItem{
			label:  dir,
			weight: g.total,
		})
	}

	squarify(outerItems, tmRect{canvasX, canvasY, canvasW, canvasH})

	// ── SVG header ───────────────────────────────────────────────────────────
	var b strings.Builder
	if _, err := fmt.Fprintf(&b, `<?xml version="1.0" encoding="UTF-8"?>
<svg width="%d" height="%d" viewBox="0 0 %d %d" xmlns="http://www.w3.org/2000/svg">
  <defs>
    <linearGradient id="riskGradient" x1="0%%" y1="0%%" x2="100%%" y2="0%%">
      <stop offset="0%%" style="stop-color:#10b981;stop-opacity:1" />
      <stop offset="50%%" style="stop-color:#f59e0b;stop-opacity:1" />
      <stop offset="100%%" style="stop-color:#ef4444;stop-opacity:1" />
    </linearGradient>
    <style>
      .cell { transition: opacity 0.15s; }
      .cell:hover { opacity: 0.85; cursor: pointer; }
      .lbl { pointer-events: none; dominant-baseline: middle; }
    </style>
  </defs>

  <!-- Background -->
  <rect width="100%%" height="100%%" fill="#0d1117" rx="10"/>

`,
		svgW, svgH, svgW, svgH,
	); err != nil {
		return err
	}

	// ── draw each directory group ─────────────────────────────────────────────
	for i, dir := range dirOrder {
		g := dirMap[dir]
		outerRect := outerItems[i].rect

		// Shrink by gap so groups have visible separation
		gr := tmRect{
			x: outerRect.x + gap,
			y: outerRect.y + gap,
			w: outerRect.w - 2*gap,
			h: outerRect.h - 2*gap,
		}
		if gr.w <= 0 || gr.h <= 0 {
			continue
		}

		// Directory backdrop
		if _, err := fmt.Fprintf(&b, `  <rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="#161b22" rx="4"/>
`,
			gr.x, gr.y, gr.w, gr.h); err != nil {
			return err
		}

		// Directory header band
		dirLabel := dir
		if dirLabel == "" {
			dirLabel = "(root)"
		}
		if gr.h > float64(hdrH)+4 {
			if _, err := fmt.Fprintf(&b, `  <rect x="%.1f" y="%.1f" width="%.1f" height="%d" fill="#21262d" rx="4"/>
`,
				gr.x, gr.y, gr.w, hdrH); err != nil {
				return err
			}
			if gr.w > 30 {
				if _, err := fmt.Fprintf(&b, `  <text x="%.1f" y="%.1f" fill="#8b949e" font-family="system-ui,-apple-system,sans-serif" font-size="10" font-weight="600" class="lbl">%s/</text>
`,
					gr.x+6, gr.y+float64(hdrH)/2+1, htmlEscape(dirLabel)); err != nil {
					return err
				}
			}
		}

		// Inner canvas for file cells (below the header band)
		innerY := gr.y + float64(hdrH) + gap
		innerH := gr.h - float64(hdrH) - gap
		if gr.h <= float64(hdrH)+4 {
			innerY = gr.y
			innerH = gr.h
		}
		innerRect := tmRect{gr.x + gap, innerY, gr.w - 2*gap, innerH}
		if innerRect.w <= 0 || innerRect.h <= 0 {
			continue
		}

		// Layout file items within this group
		items := make([]*tmItem, len(g.items))
		copy(items, g.items)
		// sort files by weight desc for better squarification
		sort.Slice(items, func(a, b int) bool { return items[a].weight > items[b].weight })

		squarify(items, innerRect)
		// Render each file cell
		for _, it := range items {
			if err := p.renderFileCell(&b, it, maxScore, minW, minH); err != nil {
				return err
			}
		}
	}

	if _, err := fmt.Fprintf(&b, `</svg>`); err != nil {
		return err
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// ─── Squarified treemap algorithm ────────────────────────────────────────────
// Reference: Bruls, Huizing, van Wijk, "Squarified Treemaps" (2000).

// squarify lays out items within rect, filling items[i].rect for every item.
func squarify(items []*tmItem, rect tmRect) {
	if len(items) == 0 || rect.w <= 0 || rect.h <= 0 {
		return
	}

	totalWeight := 0.0
	for _, it := range items {
		totalWeight += it.weight
	}
	if totalWeight == 0 {
		return
	}

	layout(items, rect, totalWeight)
}

// layout is the recursive squarify layout function.
func layout(items []*tmItem, rect tmRect, total float64) {
	if len(items) == 0 || rect.w <= 0 || rect.h <= 0 {
		return
	}
	if len(items) == 1 {
		items[0].rect = rect
		return
	}

	// Determine shorter side
	horiz := rect.w >= rect.h
	stripe := rect.h
	if horiz {
		stripe = rect.w
	}
	_ = stripe

	// Find best row using squarify criterion
	best := []int{0}
	bestRatio := worstRatio(items[:1], rect, total)

	for k := 2; k <= len(items); k++ {
		r := worstRatio(items[:k], rect, total)
		if r <= bestRatio {
			bestRatio = r
			best = make([]int, k)
			for i := range best {
				best[i] = i
			}
		} else {
			break
		}
	}

	// Place the best row
	rowItems := items[:len(best)]
	rest := items[len(best):]

	rowWeight := 0.0
	for _, it := range rowItems {
		rowWeight += it.weight
	}

	var rowRect, remainRect tmRect
	frac := rowWeight / total

	if rect.w >= rect.h {
		// horizontal strip on the left
		stripW := rect.w * frac
		rowRect = tmRect{rect.x, rect.y, stripW, rect.h}
		remainRect = tmRect{rect.x + stripW, rect.y, rect.w - stripW, rect.h}
	} else {
		// horizontal strip on the top
		stripH := rect.h * frac
		rowRect = tmRect{rect.x, rect.y, rect.w, stripH}
		remainRect = tmRect{rect.x, rect.y + stripH, rect.w, rect.h - stripH}
	}

	placeStrip(rowItems, rowRect, rowWeight)

	if len(rest) > 0 {
		restTotal := total - rowWeight
		layout(rest, remainRect, restTotal)
	}
}

// placeStrip assigns rects within a single strip.
func placeStrip(items []*tmItem, rect tmRect, total float64) {
	if rect.w >= rect.h {
		// stack items vertically within a vertical strip
		cur := rect.y
		for _, it := range items {
			h := rect.h * (it.weight / total)
			it.rect = tmRect{rect.x, cur, rect.w, h}
			cur += h
		}
	} else {
		// stack items horizontally within a horizontal strip
		cur := rect.x
		for _, it := range items {
			w := rect.w * (it.weight / total)
			it.rect = tmRect{cur, rect.y, w, rect.h}
			cur += w
		}
	}
}

// worstRatio computes the worst aspect ratio of a candidate row.
func worstRatio(items []*tmItem, rect tmRect, total float64) float64 {
	rowWeight := 0.0
	for _, it := range items {
		rowWeight += it.weight
	}
	frac := rowWeight / total
	var stripLen float64
	if rect.w >= rect.h {
		stripLen = rect.h
	} else {
		stripLen = rect.w
	}
	var stripW float64
	if rect.w >= rect.h {
		stripW = rect.w * frac
	} else {
		stripW = rect.h * frac
	}

	worst := 0.0
	for _, it := range items {
		h := stripLen * (it.weight / rowWeight)
		var r float64
		switch {
		case stripW == 0 || h == 0:
			r = math.MaxFloat64
		case stripW > h:
			r = stripW / h
		default:
			r = h / stripW
		}
		if r > worst {
			worst = r
		}
	}
	return worst
}

// ─── Visual helpers ───────────────────────────────────────────────────────────

// scoreToHex maps a normalised score [0,1] to a green→yellow→red hex colour.
func scoreToHex(norm float64) string {
	norm = math.Max(0, math.Min(1, norm))
	var r, g, bl int
	if norm < 0.5 {
		// green (#1a9e5c) → amber (#d97706)
		t := norm * 2
		r = lerp(0x1a, 0xd9, t)
		g = lerp(0x9e, 0x77, t)
		bl = lerp(0x5c, 0x06, t)
	} else {
		// amber (#d97706) → crimson (#dc2626)
		t := (norm - 0.5) * 2
		r = lerp(0xd9, 0xdc, t)
		g = lerp(0x77, 0x26, t)
		bl = lerp(0x06, 0x26, t)
	}
	return fmt.Sprintf("#%02x%02x%02x", r, g, bl)
}

func lerp(a, b int, t float64) int {
	return int(float64(a) + (float64(b)-float64(a))*t)
}

// labelFontSize picks a font size that fits within the cell.
func labelFontSize(w, h float64) float64 {
	// Scale with box size:
	// - w/7.5: ensures ~12-15 characters fit horizontally before truncation.
	// - h/2.2: ensures vertical breathing room (text height vs cell height).
	fs := math.Min(w/7.5, h/2.2)
	fs = math.Min(fs, 24) // cap at 24pt for dominant cells
	return math.Max(fs, 0)
}

// truncLabel shortens a label to at most maxChars characters.
func truncLabel(s string, maxChars int) string {
	if maxChars <= 0 {
		return ""
	}
	if len(s) <= maxChars {
		return s
	}
	if maxChars <= 3 {
		return s[:maxChars]
	}
	return s[:maxChars-1] + "…"
}

// htmlEscape escapes characters special in SVG/XML.
func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// ─── Mode metadata ────────────────────────────────────────────────────────────

// dirGroup holds items belonging to the same top-level directory.
type dirGroup struct {
	dir   string
	items []*tmItem
	total float64
}

// groupFilesByDirectory organises file results into directory-based groups.
func (p *HeatmapProvider) groupFilesByDirectory(files []schema.FileResult) (map[string]*dirGroup, []string) {
	dirMap := map[string]*dirGroup{}
	var dirOrder []string

	for _, f := range files {
		parts := strings.SplitN(filepath.ToSlash(filepath.Clean(f.Path)), "/", 2)
		var dir, name string
		if len(parts) == 2 {
			dir = parts[0]
			name = filepath.Base(f.Path)
		} else {
			dir = "" // root-level file
			name = parts[0]
		}
		score := f.ModeScore
		// Weight proportional to score² to visually amplify the difference between
		// high-risk (huge blocks) and low-risk (tiny blocks) areas.
		w := math.Max(score*score, 0.1)

		if _, ok := dirMap[dir]; !ok {
			dirMap[dir] = &dirGroup{dir: dir}
			dirOrder = append(dirOrder, dir)
		}
		g := dirMap[dir]
		g.items = append(g.items, &tmItem{
			label:    name,
			fullPath: f.Path,
			score:    score,
			weight:   w,
		})
		g.total += w
	}
	return dirMap, dirOrder
}

// renderFileCell writes a single file cell SVG element.
func (p *HeatmapProvider) renderFileCell(w io.Writer, it *tmItem, maxScore, minW, minH float64) error {
	r := it.rect
	if r.w <= 0 || r.h <= 0 {
		return nil
	}
	norm := math.Min(it.score/maxScore, 1.0)
	fill := scoreToHex(norm)

	// Slightly inset each cell to create a visual "gap" between files.
	cx, cy, cw, ch := r.x+1, r.y+1, r.w-2, r.h-2
	if cw <= 0 || ch <= 0 {
		return nil
	}

	if _, err := fmt.Fprintf(w, `  <rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s" rx="2" class="cell">
    <title>%s (Score: %.1f)</title>
  </rect>
`,
		cx, cy, cw, ch, fill, htmlEscape(it.fullPath), it.score); err != nil {
		return err
	}

	// Centred label — only if cell is large enough
	if cw < minW || ch < minH {
		return nil
	}

	lx, ly := cx+cw/2, cy+ch/2
	fs := labelFontSize(cw, ch)
	if fs < 7 {
		return nil
	}

	txt := it.label
	if ch < 20 || cw < float64(len(txt))*fs*0.55 {
		txt = truncLabel(txt, int(cw/(fs*0.5)))
	}
	if txt == "" {
		return nil
	}

	// Dark or light text for contrast
	textCol := "#0d1117"
	if norm < 0.55 {
		textCol = "#f0f6fc"
	}
	_, err := fmt.Fprintf(w, `  <text x="%.1f" y="%.1f" text-anchor="middle" fill="%s" font-family="system-ui,-apple-system,sans-serif" font-size="%.1f" class="lbl">%s</text>
`,
		lx, ly, textCol, fs, htmlEscape(txt))
	return err
}

// TreeNode is retained for backward-compatibility (unused by the new layout).
type TreeNode struct {
	Name     string
	Path     string
	Score    float64
	Size     int64
	Children map[string]*TreeNode
	IsFile   bool
}
