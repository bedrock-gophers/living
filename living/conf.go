package living

import (
	"slices"
	"time"

	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// MovementSimulator replaces the default MovementComputer for a Living entity.
// Implementations run synchronously in the entity's world transaction.
type MovementSimulator interface {
	SimulateMovement(entity *Living, tx *world.Tx, tick int64) SimulatedMovement
	SetVelocity(velocity mgl64.Vec3)
}

// SimulatedMovement is the collision-resolved movement produced for one tick.
type SimulatedMovement struct {
	Delta, Velocity mgl64.Vec3
	OnGround        bool
}

// KnockbackCalculator calculates the velocity produced by a knockback impulse.
// Returning false leaves the entity's velocity unchanged.
type KnockbackCalculator interface {
	CalculateKnockback(current, position, source mgl64.Vec3, force, height float64) (mgl64.Vec3, bool)
}

// KnockbackMotionTracker may be implemented by calculators that combine
// server-issued impulses within a tick. SetVelocity observes each impulse and
// ResetKnockbackMotion is called after the entity tick completes.
type KnockbackMotionTracker interface {
	SetVelocity(velocity mgl64.Vec3)
	ResetKnockbackMotion()
}

type Config struct {
	world.EntityType
	*entity.MovementComputer
	MovementSimulator           MovementSimulator
	Knockback                   KnockbackCalculator
	Attachment                  any
	Speed, EyeHeight, MaxHealth float64
	Drops                       []Drop
	ImmuneDuration              time.Duration
	Handler
}

func (c Config) Apply(data *world.EntityData) {
	if c.EntityType == nil {
		panic("entity type can't be nil")
	}

	if c.Handler == nil {
		c.Handler = NopHandler{}
	}

	data.Data = &livingData{
		entityType:     c.EntityType,
		mc:             c.MovementComputer,
		movement:       c.MovementSimulator,
		knockback:      c.Knockback,
		attachment:     c.Attachment,
		speed:          c.Speed,
		eyeHeight:      c.EyeHeight,
		HealthManager:  entity.NewHealthManager(c.MaxHealth, c.MaxHealth),
		drops:          slices.Values(c.Drops),
		scale:          1,
		immuneDuration: c.ImmuneDuration,
		effects:        make(map[effect.Type]effect.Effect),
		handler:        c.Handler,
	}
}
