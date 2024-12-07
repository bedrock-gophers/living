package living

import (
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/world"
	"time"
)

type livingData struct {
	*entity.HealthManager

	drops      []Drop
	entityType world.EntityType

	mc *entity.MovementComputer

	collidedHorizontally bool
	collidedVertically   bool

	onGround bool
	immobile bool

	fallDistance float64
	fireTicks    int64

	immuneUntil time.Time
	lastDamage  float64
	speed       float64

	handler Handler
}
