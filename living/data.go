package living

import (
	"github.com/df-mc/dragonfly/server/entity"
	"time"
)

type livingData struct {
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
}
