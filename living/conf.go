package living

import (
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

type Config struct {
	Position mgl64.Vec3
}

func (c Config) Apply(data *world.EntityData) {
	data.Data = &livingData{
		mc:    &entity.MovementComputer{Gravity: 0.08, Drag: 0.02, DragBeforeGravity: true},
		speed: 0.1,
	}
}
