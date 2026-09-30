package storage

import d "tramflow/internal/domain"

// Hash binds the whole artifact, including any offline preprocessing metadata.
func featureSchemaClaim(model d.Model) any {
	return struct {
		Version string   `json:"version"`
		SHA256  string   `json:"sha256"`
		Columns []string `json:"columns"`
	}{Version: model.Schema, SHA256: model.SchemaSHA256, Columns: model.Columns}
}
