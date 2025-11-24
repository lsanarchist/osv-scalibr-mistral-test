package components_test

import (
	"strings"
	"testing"

	"deps.dev/util/resolve"
	"github.com/google/osv-scalibr/guidedremediation/internal/resolution"
	"github.com/google/osv-scalibr/guidedremediation/internal/tui/components"
)

func TestFindChainGraphs(t *testing.T) {
	// Graph: Root -> A -> B (Vuln)
	// Node IDs: Root=0, A=1, B=2
	// Distances are to the vulnerable node B.
	subgraph := &resolution.DependencySubgraph{
		Dependency: 2,
		Nodes: map[resolve.NodeID]resolution.GraphNode{
			0: {
				Version: resolve.VersionKey{
					PackageKey: resolve.PackageKey{Name: "root", System: resolve.NPM},
					Version:    "1.0.0",
				},
				Distance: 2,
				Children: []resolve.Edge{{From: 0, To: 1}},
			},
			1: {
				Version: resolve.VersionKey{
					PackageKey: resolve.PackageKey{Name: "pkgA", System: resolve.NPM},
					Version:    "1.0.0",
				},
				Distance: 1,
				Parents:  []resolve.Edge{{From: 0, To: 1}},
				Children: []resolve.Edge{{From: 1, To: 2}},
			},
			2: {
				Version: resolve.VersionKey{
					PackageKey: resolve.PackageKey{Name: "pkgB", System: resolve.NPM},
					Version:    "1.0.0",
				},
				Distance: 0,
				Parents:  []resolve.Edge{{From: 1, To: 2}},
			},
		},
	}

	graphs := components.FindChainGraphs([]*resolution.DependencySubgraph{subgraph})

	if len(graphs) != 1 {
		t.Fatalf("Expected 1 chain graph, got %d", len(graphs))
	}

	output := graphs[0].String()
	// We expect pkgA to be above pkgB
	if !strings.Contains(output, "pkgA@1.0.0") {
		t.Errorf("Expected output to contain pkgA@1.0.0, got:\n%s", output)
	}
	if !strings.Contains(output, "pkgB@1.0.0") {
		t.Errorf("Expected output to contain pkgB@1.0.0, got:\n%s", output)
	}
}

func TestFindChainGraphs_Branching(t *testing.T) {
	// Graph:
	// Root -> A -> B (Vuln)
	// Root -> C -> B (Vuln)
	// Node IDs: Root=0, A=1, B=2, C=3
	// Distances to B.
	subgraph := &resolution.DependencySubgraph{
		Dependency: 2,
		Nodes: map[resolve.NodeID]resolution.GraphNode{
			0: {
				Version: resolve.VersionKey{
					PackageKey: resolve.PackageKey{Name: "root", System: resolve.NPM},
					Version:    "1.0.0",
				},
				Distance: 2,
				Children: []resolve.Edge{{From: 0, To: 1}, {From: 0, To: 3}},
			},
			1: {
				Version: resolve.VersionKey{
					PackageKey: resolve.PackageKey{Name: "pkgA", System: resolve.NPM},
					Version:    "1.0.0",
				},
				Distance: 1,
				Parents:  []resolve.Edge{{From: 0, To: 1}},
				Children: []resolve.Edge{{From: 1, To: 2}},
			},
			2: {
				Version: resolve.VersionKey{
					PackageKey: resolve.PackageKey{Name: "pkgB", System: resolve.NPM},
					Version:    "1.0.0",
				},
				Distance: 0,
				Parents:  []resolve.Edge{{From: 1, To: 2}, {From: 3, To: 2}},
			},
			3: {
				Version: resolve.VersionKey{
					PackageKey: resolve.PackageKey{Name: "pkgC", System: resolve.NPM},
					Version:    "1.0.0",
				},
				Distance: 1,
				Parents:  []resolve.Edge{{From: 0, To: 3}},
				Children: []resolve.Edge{{From: 3, To: 2}},
			},
		},
	}

	graphs := components.FindChainGraphs([]*resolution.DependencySubgraph{subgraph})

	if len(graphs) != 1 {
		t.Fatalf("Expected 1 chain graph, got %d", len(graphs))
	}

	output := graphs[0].String()
	if !strings.Contains(output, "pkgA@1.0.0") {
		t.Errorf("Expected output to contain pkgA@1.0.0, got:\n%s", output)
	}
	if !strings.Contains(output, "pkgC@1.0.0") {
		t.Errorf("Expected output to contain pkgC@1.0.0, got:\n%s", output)
	}
	if !strings.Contains(output, "pkgB@1.0.0") {
		t.Errorf("Expected output to contain pkgB@1.0.0, got:\n%s", output)
	}
}

func TestFindChainGraphs_DirectVuln(t *testing.T) {
	// Graph: Root -> A (Vuln)
	// Node IDs: Root=0, A=1
	subgraph := &resolution.DependencySubgraph{
		Dependency: 1,
		Nodes: map[resolve.NodeID]resolution.GraphNode{
			0: {
				Version: resolve.VersionKey{
					PackageKey: resolve.PackageKey{Name: "root", System: resolve.NPM},
					Version:    "1.0.0",
				},
				Distance: 1,
				Children: []resolve.Edge{{From: 0, To: 1}},
			},
			1: {
				Version: resolve.VersionKey{
					PackageKey: resolve.PackageKey{Name: "pkgA", System: resolve.NPM},
					Version:    "1.0.0",
				},
				Distance: 0,
				Parents:  []resolve.Edge{{From: 0, To: 1}},
			},
		},
	}

	graphs := components.FindChainGraphs([]*resolution.DependencySubgraph{subgraph})

	if len(graphs) != 1 {
		t.Fatalf("Expected 1 chain graph, got %d", len(graphs))
	}

	output := graphs[0].String()
	if !strings.Contains(output, "pkgA@1.0.0") {
		t.Errorf("Expected output to contain pkgA@1.0.0, got:\n%s", output)
	}
}
