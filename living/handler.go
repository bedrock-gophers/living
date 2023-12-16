package living

import (
	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/world"
)

type Handler interface {
	// HandleTick handles the entity's tick.
	HandleTick()
	// HandleHurt handles the entity being hurt.
	HandleHurt(ctx *event.Context, damage float64, src world.DamageSource)
}

type NopHandler struct{}

var _ Handler = NopHandler{}

func (NopHandler) HandleTick() {}

func (NopHandler) HandleHurt(*event.Context, float64, world.DamageSource) {
}
