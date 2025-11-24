package model

import (
	"testing"

	"github.com/google/osv-scalibr/guidedremediation/internal/remediation"
	"github.com/google/osv-scalibr/guidedremediation/internal/resolution"
	"github.com/google/osv-scalibr/guidedremediation/internal/strategy/common"
	"github.com/google/osv-scalibr/guidedremediation/result"
)

func TestNewStateRelockResult(t *testing.T) {
	m := Model{
		relockBaseManifest: &remediation.ResolvedManifest{
			ResolvedGraph: remediation.ResolvedGraph{
				Vulns: []resolution.Vulnerability{},
			},
		},
		relockBaseErrors: []result.ResolveError{},
		viewWidth:        80,
		viewHeight:       24,
	}

	st := newStateRelockResult(m)

	if st.currRes != m.relockBaseManifest {
		t.Errorf("Expected currRes to be %v, got %v", m.relockBaseManifest, st.currRes)
	}
	if len(st.currErrs) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(st.currErrs))
	}
	if st.patchesDone {
		t.Errorf("Expected patchesDone to be false")
	}
}

func TestStateRelockResult_GetEffectiveCursor(t *testing.T) {
	tests := []struct {
		name       string
		cursorPos  int
		patches    common.PatchResult
		wantCursor relockCursorPos
	}{
		{
			name:       "Before patches",
			cursorPos:  int(relockRemaining),
			wantCursor: relockRemaining,
		},
		{
			name:       "At patches start",
			cursorPos:  int(relockPatches),
			patches:    common.PatchResult{Patches: []result.Patch{{}}},
			wantCursor: relockPatches,
		},
		{
			name:       "Inside patches",
			cursorPos:  int(relockPatches) + 1,
			patches:    common.PatchResult{Patches: []result.Patch{{}, {}}},
			wantCursor: relockPatches,
		},
		{
			name:       "After patches",
			cursorPos:  int(relockPatches) + 2,
			patches:    common.PatchResult{Patches: []result.Patch{{}, {}}},
			wantCursor: relockApply, // relockPatches + 1 (apply)
		},
		{
			name:       "No patches, skip to write",
			cursorPos:  int(relockPatches),
			patches:    common.PatchResult{Patches: []result.Patch{}},
			wantCursor: relockWrite, // relockPatches + 2 (write)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := stateRelockResult{
				cursorPos: tt.cursorPos,
				patches:   tt.patches,
			}
			if got := st.getEffectiveCursor(); got != tt.wantCursor {
				t.Errorf("getEffectiveCursor() = %v, want %v", got, tt.wantCursor)
			}
		})
	}
}

func TestStateRelockResult_PatchCompatible(t *testing.T) {
	// Setup patches
	p1 := result.Patch{
		PackageUpdates: []result.PackageUpdate{
			{Name: "pkgA"},
		},
	}
	p2 := result.Patch{
		PackageUpdates: []result.PackageUpdate{
			{Name: "pkgB"},
		},
	}
	p3 := result.Patch{
		PackageUpdates: []result.PackageUpdate{
			{Name: "pkgA"}, // Conflicts with p1
		},
	}

	st := stateRelockResult{
		patches: common.PatchResult{
			Patches: []result.Patch{p1, p2, p3},
		},
		selectedPatches: make(map[int]struct{}),
	}

	// Select p1
	st.selectedPatches[0] = struct{}{}

	if !st.patchCompatible(0) {
		t.Errorf("Selected patch should be compatible with itself")
	}
	if !st.patchCompatible(1) {
		t.Errorf("Independent patch should be compatible")
	}
	if st.patchCompatible(2) {
		t.Errorf("Conflicting patch should not be compatible")
	}
}
