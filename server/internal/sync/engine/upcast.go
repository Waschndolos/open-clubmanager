package engine

import (
	"fmt"

	"github.com/Waschndolos/open-clubmanager/server/internal/sync/op"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/segment"
)

// upcasters[n] converts ops of schema version n to version n+1 (§16): rename
// columns, set defaults, drop removed columns. Every migration that changes
// a synced table raises segment.SchemaVersion and adds one here.
var upcasters = map[int]func([]op.Op) ([]op.Op, error){}

// upcastOps converts ops written with schema version from to the local
// version (§13.3 step 1). Callers have rejected versions above the local one.
func upcastOps(from int, ops []op.Op) ([]op.Op, error) {
	for v := from; v < segment.SchemaVersion; v++ {
		up, ok := upcasters[v]
		if !ok {
			return nil, fmt.Errorf("no upcaster from schema %d", v)
		}
		var err error
		if ops, err = up(ops); err != nil {
			return nil, err
		}
	}
	return ops, nil
}
