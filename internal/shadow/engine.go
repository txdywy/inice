package shadow

import (
	"context"

	"github.com/txdywy/inice/internal/model"
)

// Engine defines a core engine (xray, sing-box, hysteria, etc.)
type Engine interface {
	// Name returns the engine name (e.g., "xray")
	Name() string
	
	// CanHandle returns true if this engine should handle the given node type
	CanHandle(node model.ProxyNode) bool
	
	// Setup generates config(s), uploads them, and starts the process(es) on the router.
	// It returns a list of cleanup functions or an error.
	Setup(ctx context.Context, nodes []model.ProxyNode) (cleanupFuncs []func(context.Context) error, err error)
}
