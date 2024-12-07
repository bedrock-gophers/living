package living

import (
	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/world"
	"time"
)

type Context = event.Context[*Living]

type Handler interface {
	// HandleTick handles the entity's tick.
	HandleTick(ctx *Context)
	// HandleHurt handles the entity being hurt.
	HandleHurt(ctx *Context, damage float64, immune bool, immunity *time.Duration, src world.DamageSource)
}

type NopHandler struct{}

var _ Handler = NopHandler{}

func (NopHandler) HandleTick(*Context) {}

func (NopHandler) HandleHurt(*Context, float64, bool, *time.Duration, world.DamageSource) {
}
