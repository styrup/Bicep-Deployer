package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestParseModuleRefs(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "no modules",
			content: "param location string = 'westeurope'\nresource rg 'Microsoft.Resources/resourceGroups@2023-07-01' = {\n",
			want:    []string{},
		},
		{
			name:    "single module",
			content: "module budget './modules/budget.bicep' = {\n  name: 'budgetModule'\n}\n",
			want:    []string{"./modules/budget.bicep"},
		},
		{
			name:    "conditional module",
			content: "module budget './modules/budget.bicep' = if (monthlyBudgetUSD > 0) {\n  name: 'budgetModule'\n}\n",
			want:    []string{"./modules/budget.bicep"},
		},
		{
			name: "multiple modules",
			content: `module net './networking/vnet.bicep' = {
  name: 'vnet'
}
module app './compute/app.bicep' = {
  name: 'app'
}
`,
			want: []string{"./networking/vnet.bicep", "./compute/app.bicep"},
		},
		{
			name:    "registry reference ignored",
			content: "module storage 'br:myregistry.azurecr.io/bicep/storage:v1' = {\n}\n",
			want:    []string{},
		},
		{
			name:    "template spec ignored",
			content: "module storage 'ts:00000000-0000-0000-0000-000000000000/rg/spec:v1' = {\n}\n",
			want:    []string{},
		},
		{
			name:    "indented module",
			content: "  module helper '../shared/helper.bicep' = {\n}\n",
			want:    []string{"../shared/helper.bicep"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseModuleRefs(tt.content)
			if len(got) != len(tt.want) {
				t.Fatalf("parseModuleRefs() returned %d refs, want %d: %v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ref[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// fakeStore is a simple in-memory TemplateStore for testing.
type fakeStore struct {
	files map[string]string
}

func (f *fakeStore) ListTemplates(_ context.Context) ([]string, error) {
	names := make([]string, 0, len(f.files))
	for k := range f.files {
		names = append(names, k)
	}
	return names, nil
}

func (f *fakeStore) DownloadTemplate(_ context.Context, name string) (string, error) {
	content, ok := f.files[name]
	if !ok {
		return "", fmt.Errorf("not found: %s", name)
	}
	return content, nil
}

func TestDownloadModules(t *testing.T) {
	store := &fakeStore{files: map[string]string{
		"governance/resource-group.bicep": "module budget './modules/budget.bicep' = if (cond) {\n}\n",
		"governance/modules/budget.bicep": "param amount int\nresource b 'Microsoft.Consumption/budgets@2023-05-01' = {\n}\n",
	}}

	tmpDir := t.TempDir()

	mainContent := store.files["governance/resource-group.bicep"]
	mainPath := filepath.Join(tmpDir, "governance", "resource-group.bicep")
	if err := writeFile(mainPath, mainContent); err != nil {
		t.Fatalf("write main template: %v", err)
	}

	err := downloadModules(context.Background(), store, tmpDir, "governance/resource-group.bicep", mainContent)
	if err != nil {
		t.Fatalf("downloadModules() error: %v", err)
	}

	// Verify the module was downloaded to the correct relative path.
	modulePath := filepath.Join(tmpDir, "governance", "modules", "budget.bicep")
	data, err := os.ReadFile(modulePath)
	if err != nil {
		t.Fatalf("module file not found at %s: %v", modulePath, err)
	}

	if string(data) != store.files["governance/modules/budget.bicep"] {
		t.Errorf("module content mismatch:\ngot:  %q\nwant: %q", string(data), store.files["governance/modules/budget.bicep"])
	}
}

func TestDownloadModulesNested(t *testing.T) {
	store := &fakeStore{files: map[string]string{
		"main.bicep":            "module a './modules/a.bicep' = {\n}\n",
		"modules/a.bicep":       "module b './nested/b.bicep' = {\n}\n",
		"modules/nested/b.bicep": "param x string\n",
	}}

	tmpDir := t.TempDir()
	mainContent := store.files["main.bicep"]
	if err := writeFile(filepath.Join(tmpDir, "main.bicep"), mainContent); err != nil {
		t.Fatal(err)
	}

	err := downloadModules(context.Background(), store, tmpDir, "main.bicep", mainContent)
	if err != nil {
		t.Fatalf("downloadModules() error: %v", err)
	}

	// Both module a and nested module b should exist.
	for _, rel := range []string{"modules/a.bicep", "modules/nested/b.bicep"} {
		p := filepath.Join(tmpDir, filepath.FromSlash(rel))
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected file %s not found: %v", rel, err)
		}
	}
}
