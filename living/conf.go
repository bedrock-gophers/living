package living

import (
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/world"
)

type Config struct {
	EntityType world.EntityType
	Handler    Handler
	Drops      []Drop
}

func (c Config) Apply(data *world.EntityData) {
	if c.EntityType == nil {
		panic("entity type can't be nil")
	}

	if c.Handler == nil {
		c.Handler = NopHandler{}
	}

	data.Data = &livingData{
		drops:         c.Drops,
		entityType:    c.EntityType,
		HealthManager: entity.NewHealthManager(20, 20),
		mc:            &entity.MovementComputer{Gravity: 0.08, Drag: 0.02, DragBeforeGravity: true},
		speed:         0.1,
		handler:       c.Handler,
	}
}
