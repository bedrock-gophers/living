package living

import (
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/world"
	"iter"
	"time"
)

type livingData struct {
	entityType world.EntityType

	age       time.Duration
	mc        *entity.MovementComputer
	speed     float64
	eyeHeight float64
	*entity.HealthManager

	drops iter.Seq[Drop]

	collidedHorizontally bool
	collidedVertically   bool

	onGround  bool
	immobile  bool
	invisible bool
	scale     float64

	fallDistance float64
	fireTicks    int64

	immuneUntil    time.Time
	immuneDuration time.Duration
	lastDamage     float64

	effects map[effect.Type]effect.Effect

	handler Handler

	variant     int32
	markVariant int32
}
