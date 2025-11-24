package model

import (
	"testing"

	"github.com/google/osv-scalibr/guidedremediation/internal/remediation"
	"github.com/google/osv-scalibr/guidedremediation/internal/resolution"
	"github.com/google/osv-scalibr/guidedremediation/options"
	"github.com/google/osv-scalibr/guidedremediation/result"
)

func TestNewStateChooseStrategy(t *testing.T) {
	m := Model{
		lockfileGraph: remediation.ResolvedGraph{
			Vulns: []resolution.Vulnerability{},
		},
		lockfilePatches: []result.Patch{},
		options: options.FixVulnsOptions{
			Manifest: "package.json",
			RemediationOptions: options.RemediationOptions{
				MaxDepth: 5,
			},
		},
		relockBaseManifest: &remediation.ResolvedManifest{
			ResolvedGraph: remediation.ResolvedGraph{
				Vulns: []resolution.Vulnerability{},
			},
		},
	}

	st := newStateChooseStrategy(m)

	if st.cursorPos != chooseStratInPlace {
		t.Errorf("Expected cursorPos to be %v, got %v", chooseStratInPlace, st.cursorPos)
	}
	if !st.canRelock {
		t.Errorf("Expected canRelock to be true")
	}
	if st.depthInput.Value() != "5" {
		t.Errorf("Expected depthInput to be 5, got %s", st.depthInput.Value())
	}
}

func TestNewStateChooseStrategy_NoManifest(t *testing.T) {
	m := Model{
		lockfileGraph: remediation.ResolvedGraph{
			Vulns: []resolution.Vulnerability{},
		},
		lockfilePatches: []result.Patch{},
		options: options.FixVulnsOptions{
			Manifest: "", // No manifest
		},
	}

	st := newStateChooseStrategy(m)

	if st.canRelock {
		t.Errorf("Expected canRelock to be false")
	}
}
