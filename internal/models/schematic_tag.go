package models

type SchematicTag struct {
	ID   string
	Key  string
	Name string
}

type SchematicTagWithCount struct {
	ID    string
	Key   string
	Name  string
	Count int64
}
