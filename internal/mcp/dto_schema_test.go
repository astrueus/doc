package mcp_test

import (
	"testing"

	"git.itopcms.com/astrueus/doc/internal/dto/mcpdto"
	"github.com/google/jsonschema-go/jsonschema"
)

func TestMCPDTOInputSchemas(t *testing.T) {
	checks := []struct {
		name string
		fn   func() (*jsonschema.Schema, error)
	}{
		{"SearchDocumentIn", func() (*jsonschema.Schema, error) { return jsonschema.For[mcpdto.SearchDocumentIn](nil) }},
		{"GetDocumentIn", func() (*jsonschema.Schema, error) { return jsonschema.For[mcpdto.GetDocumentIn](nil) }},
		{"ListBooksIn", func() (*jsonschema.Schema, error) { return jsonschema.For[mcpdto.ListBooksIn](nil) }},
		{"ListDocumentTreeIn", func() (*jsonschema.Schema, error) { return jsonschema.For[mcpdto.ListDocumentTreeIn](nil) }},
		{"CreateDocumentIn", func() (*jsonschema.Schema, error) { return jsonschema.For[mcpdto.CreateDocumentIn](nil) }},
		{"UpdateDocumentContentIn", func() (*jsonschema.Schema, error) { return jsonschema.For[mcpdto.UpdateDocumentContentIn](nil) }},
		{"AppendDocumentContentIn", func() (*jsonschema.Schema, error) { return jsonschema.For[mcpdto.AppendDocumentContentIn](nil) }},
		{"UpdateDocumentMetaIn", func() (*jsonschema.Schema, error) { return jsonschema.For[mcpdto.UpdateDocumentMetaIn](nil) }},
		{"ReleaseDocumentIn", func() (*jsonschema.Schema, error) { return jsonschema.For[mcpdto.ReleaseDocumentIn](nil) }},
		{"DeleteDocumentIn", func() (*jsonschema.Schema, error) { return jsonschema.For[mcpdto.DeleteDocumentIn](nil) }},
	}
	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			s, err := c.fn()
			if err != nil {
				t.Fatalf("schema: %v", err)
			}
			if s == nil {
				t.Fatal("nil schema")
			}
		})
	}
}
