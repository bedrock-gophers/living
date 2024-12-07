package living

import (
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/world"
)

type Config struct {
	EntityType world.EntityType
}

func (c Config) Apply(data *world.EntityData) {
	if c.EntityType == nil {
		panic("entity type can't be nil")
	}
	data.Data = &livingData{
		entityType: c.EntityType,
		mc:         &entity.MovementComputer{Gravity: 0.08, Drag: 0.02, DragBeforeGravity: true},
		speed:      0.1,
	}
}
