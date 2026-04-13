package bicep

import (
	"testing"
)

func TestParseMultiLineObjectDefault(t *testing.T) {
	source := `@description('Tags')
param tags object = {
  environment: 'dev'
  managedBy: 'bicep-deployer'
}
`
	params := ParseParameters(source)
	if len(params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(params))
	}
	p := params[0]
	if p.Name != "tags" {
		t.Errorf("name = %q, want %q", p.Name, "tags")
	}
	if p.Type != TypeObject {
		t.Errorf("type = %q, want %q", p.Type, TypeObject)
	}
	if p.Required {
		t.Error("expected Required = false")
	}
	if p.DefaultValue == nil {
		t.Fatal("expected DefaultValue to be set")
	}
	if val := *p.DefaultValue; val == "{" {
		t.Errorf("DefaultValue should not be just '{', got %q", val)
	}
	if p.Description != "Tags" {
		t.Errorf("description = %q, want %q", p.Description, "Tags")
	}
}

func TestParseMultiLineArrayDefault(t *testing.T) {
	source := `param zones array = [
  '1'
  '2'
  '3'
]
`
	params := ParseParameters(source)
	if len(params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(params))
	}
	p := params[0]
	if p.Name != "zones" {
		t.Errorf("name = %q, want %q", p.Name, "zones")
	}
	if p.Type != TypeArray {
		t.Errorf("type = %q, want %q", p.Type, TypeArray)
	}
	if p.DefaultValue == nil {
		t.Fatal("expected DefaultValue to be set")
	}
	if val := *p.DefaultValue; val == "[" {
		t.Errorf("DefaultValue should not be just '[', got %q", val)
	}
}

func TestParseNestedObjectDefault(t *testing.T) {
	source := `param config object = {
  network: {
    vnetName: 'vnet-01'
  }
  tags: {
    env: 'dev'
  }
}
`
	params := ParseParameters(source)
	if len(params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(params))
	}
	p := params[0]
	if p.DefaultValue == nil {
		t.Fatal("expected DefaultValue to be set")
	}
	if val := *p.DefaultValue; val == "{" {
		t.Errorf("DefaultValue should not be just '{', got %q", val)
	}
}

func TestParseParamAfterMultiLineDefault(t *testing.T) {
	source := `param tags object = {
  env: 'dev'
}

@description('Azure region')
param location string = 'westeurope'
`
	params := ParseParameters(source)
	if len(params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(params))
	}
	if params[0].Name != "tags" {
		t.Errorf("first param name = %q, want %q", params[0].Name, "tags")
	}
	if params[1].Name != "location" {
		t.Errorf("second param name = %q, want %q", params[1].Name, "location")
	}
	if params[1].Description != "Azure region" {
		t.Errorf("second param description = %q, want %q", params[1].Description, "Azure region")
	}
	if params[1].DefaultValue == nil || *params[1].DefaultValue != "westeurope" {
		t.Errorf("second param default = %v, want 'westeurope'", params[1].DefaultValue)
	}
}

func TestParseSingleLineObjectDefault(t *testing.T) {
	source := `param tags object = {}
`
	params := ParseParameters(source)
	if len(params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(params))
	}
	if params[0].DefaultValue == nil || *params[0].DefaultValue != "{}" {
		t.Errorf("DefaultValue = %v, want '{}'", params[0].DefaultValue)
	}
}
