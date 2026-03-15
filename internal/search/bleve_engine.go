package search

import "context"

// BleveEngine wraps the existing *Service to implement SearchEngine.
// variant controls which Bleve index is used for full-text matching:
//   - "base" uses bleveBaseIndex (no AIDescription field)
//   - "ai"   uses bleveIndex (includes AIDescription — current behavior)
type BleveEngine struct {
	service  *Service
	useBase  bool // true = variant A (no AI), false = variant B (with AI)
}

// NewBleveEngine creates a SearchEngine backed by the existing Bleve service.
// If useBase is true, text searches use the base index (variant A);
// otherwise they use the full index with AI descriptions (variant B).
func NewBleveEngine(svc *Service, useBase bool) *BleveEngine {
	return &BleveEngine{service: svc, useBase: useBase}
}

func (b *BleveEngine) Search(_ context.Context, q SearchQuery) ([]string, error) {
	var ids []string
	if b.useBase {
		ids = b.service.SearchWithIndex(q.Term, q.Order, q.Rating, q.Category, q.Tags,
			q.MinecraftVersion, q.CreateVersion, q.HidePaid, true)
	} else {
		ids = b.service.Search(q.Term, q.Order, q.Rating, q.Category, q.Tags,
			q.MinecraftVersion, q.CreateVersion, q.HidePaid)
	}
	return ids, nil
}

func (b *BleveEngine) Suggest(q string, limit int) []Suggestion {
	return b.service.Suggest(q, limit)
}

func (b *BleveEngine) Ready() bool {
	return b.service.Ready()
}

func (b *BleveEngine) Health(_ context.Context) error {
	if b.service.Ready() {
		return nil
	}
	return errNotReady
}

var errNotReady = &engineError{msg: "bleve index not ready"}

type engineError struct{ msg string }

func (e *engineError) Error() string { return e.msg }
