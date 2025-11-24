package model

import (
	"testing"

	"github.com/google/osv-scalibr/guidedremediation/internal/remediation"
	"github.com/google/osv-scalibr/guidedremediation/internal/resolution"
	"github.com/google/osv-scalibr/guidedremediation/internal/tui/components"
	"github.com/google/osv-scalibr/guidedremediation/options"
	"github.com/google/osv-scalibr/guidedremediation/result"
)

func TestNewStateInPlaceResult(t *testing.T) {
	m := Model{
		lockfileGraph: remediation.ResolvedGraph{
			Vulns: []resolution.Vulnerability{},
		},
		lockfilePatches: []result.Patch{},
		options: options.FixVulnsOptions{
			Manifest: "package.json",
		},
		relockBaseManifest: &remediation.ResolvedManifest{
			ResolvedGraph: remediation.ResolvedGraph{
				Vulns: []resolution.Vulnerability{},
			},
		},
	}

	inPlaceInfo := components.TextView("info")
	selectedChanges := []bool{true, false}

	st := newStateInPlaceResult(m, inPlaceInfo, selectedChanges)

	if st.cursorPos != inPlaceChoice {
		t.Errorf("Expected cursorPos to be %v, got %v", inPlaceChoice, st.cursorPos)
	}
	if !st.canRelock {
		t.Errorf("Expected canRelock to be true")
	}
	if len(st.selectedChanges) != 2 {
		t.Errorf("Expected 2 selected changes, got %d", len(st.selectedChanges))
	}
	if !st.selectedChanges[0] || st.selectedChanges[1] {
		t.Errorf("Expected selectedChanges to be [true, false], got %v", st.selectedChanges)
	}
}

func TestNewStateInPlaceResult_NoSelection(t *testing.T) {
	m := Model{
		lockfileGraph: remediation.ResolvedGraph{
			Vulns: []resolution.Vulnerability{},
		},
		lockfilePatches: []result.Patch{
			{
				PackageUpdates: []result.PackageUpdate{{Name: "pkgA"}},
			},
			{
				PackageUpdates: []result.PackageUpdate{{Name: "pkgB"}},
			},
		},
		options: options.FixVulnsOptions{
			Manifest: "package.json",
		},
		relockBaseManifest: &remediation.ResolvedManifest{
			ResolvedGraph: remediation.ResolvedGraph{
				Vulns: []resolution.Vulnerability{},
			},
		},
	}

	inPlaceInfo := components.TextView("info")

	// Pass nil for selectedChanges
	st := newStateInPlaceResult(m, inPlaceInfo, nil)

	if len(st.selectedChanges) != 2 {
		t.Errorf("Expected 2 selected changes, got %d", len(st.selectedChanges))
	}
	// chooseAllCompatiblePatches should select all compatible patches.
	// Since these are independent, both should be selected.
	if !st.selectedChanges[0] || !st.selectedChanges[1] {
		t.Errorf("Expected all patches to be selected, got %v", st.selectedChanges)
	}
}
