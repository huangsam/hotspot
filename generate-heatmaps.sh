#!/bin/bash
# Hotspot Heatmap Demo Script
# Generates visual risk heatmaps for your repository

set -e

echo "🔥 Hotspot Heatmap Generator"
echo "============================"

# Check if local hotspot binary exists, otherwise use system one
if [ -f "./bin/hotspot" ]; then
    HOTSPOT_CMD="./bin/hotspot"
elif command -v hotspot &> /dev/null; then
    HOTSPOT_CMD="hotspot"
else
    echo "❌ Hotspot CLI not found. Please install it first:"
    echo "   go install github.com/huangsam/hotspot@latest"
    echo "   or build locally: go build -o bin/hotspot ."
    exit 1
fi

# Generate heatmaps for different risk modes
echo "📊 Generating activity heatmap..."
$HOTSPOT_CMD files --mode hot --limit 25 --output heatmap --output-file hotspot-activity-heatmap.svg

echo "🎯 Generating risk heatmap..."
$HOTSPOT_CMD files --mode risk --limit 25 --output heatmap --output-file hotspot-risk-heatmap.svg

echo "🏗️  Generating complexity heatmap..."
$HOTSPOT_CMD files --mode complexity --limit 25 --output heatmap --output-file hotspot-complexity-heatmap.svg

echo "💰 Generating ROI heatmap..."
$HOTSPOT_CMD files --mode roi --limit 25 --output heatmap --output-file hotspot-roi-heatmap.svg

echo ""
echo "✅ Heatmaps generated successfully!"
echo ""
echo "📁 Files created:"
echo "   - hotspot-activity-heatmap.svg    (Recent activity & churn)"
echo "   - hotspot-risk-heatmap.svg        (Knowledge concentration)"
echo "   - hotspot-complexity-heatmap.svg  (Size & age factors)"
echo "   - hotspot-roi-heatmap.svg         (Refactoring priority)"
echo ""
echo "🌐 Open these SVG files in your browser to explore your codebase risks!"
echo "💡 Pro tip: Share these with your team for data-driven architecture decisions."
