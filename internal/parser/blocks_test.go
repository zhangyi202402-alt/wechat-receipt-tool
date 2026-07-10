package parser

import "testing"

func TestMergeReceiptBlocks(t *testing.T) {
	blocks := []block{
		{lines: []visualLine{
			{text: "二维码收款-来自*辛", topY: 268, bottom: 285, score: 0.98},
			{text: "+30.00", topY: 268, bottom: 285, score: 0.95},
		}},
		{lines: []visualLine{
			{text: "二维码收款-来自*辛 7月9日19:47", topY: 278, bottom: 295, score: 0.97},
			{text: "7月9日19:47", topY: 290, bottom: 305, score: 0.98},
		}},
	}
	merged := mergeReceiptBlocks(blocks)
	if len(merged) != 1 {
		t.Fatalf("expected 1 merged block, got %d", len(merged))
	}
	if len(merged[0].lines) != 4 {
		t.Fatalf("expected 4 unique lines, got %d", len(merged[0].lines))
	}
}

func TestClusterBlocksSameSource(t *testing.T) {
	lines := []visualLine{
		{text: "二维码收款-来自*云", topY: 194, bottom: 210, score: 0.99},
		{text: "+2272.00", topY: 194, bottom: 210, score: 0.97},
		{text: "二维码收款-来自*云 7月9日20:21", topY: 204, bottom: 220, score: 0.98},
		{text: "7月9日20:21", topY: 214, bottom: 228, score: 0.97},
	}
	blocks := mergeReceiptBlocks(clusterBlocks(lines, defaultBlockGap))
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block for *云, got %d", len(blocks))
	}
}
