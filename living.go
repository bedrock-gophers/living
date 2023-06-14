package living

import (
	"math"
	"time"

	"github.com/df-mc/atomic"
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

type Living struct {
	w *world.World

	entityType world.EntityType

	health    float64
	maxHealth float64

	drops []item.Stack

	lastAttack time.Time
	rot        cube.Rotation

	pos mgl64.Vec3
	vel mgl64.Vec3

	speed        float64
	onGround     atomic.Bool
	fallDistance atomic.Float64

	collidedVertically, collidedHorizontally atomic.Bool

	onHurt func(damage float64, src world.DamageSource) (n float64, vulnerable bool)
	onHeal func(health float64, src world.HealingSource)

	mc *entity.MovementComputer
}

// NewLivingEntity creates a new entity based on the data provided.
func NewLivingEntity(entityType world.EntityType, maxHealth float64, speed float64, drops []item.Stack, mc *entity.MovementComputer, pos mgl64.Vec3, w *world.World) *Living {
	return &Living{entityType: entityType, health: maxHealth, maxHealth: maxHealth, drops: drops, speed: speed, mc: mc, pos: pos, w: w}
}

// Health returns the current health of the entity.
func (e *Living) Health() float64 {
	return e.health
}

// MaxHealth returns the max health of the entity.
func (e *Living) MaxHealth() float64 {
	return e.maxHealth
}

// SetMaxHealth sets the max health of the entity.
func (e *Living) SetMaxHealth(v float64) {
	e.maxHealth = v
}

// SetOnHurt sets the function called when the entity is hurt.
func (e *Living) SetOnHurt(f func(damage float64, src world.DamageSource) (n float64, vulnerable bool)) {
	e.onHurt = f
}

// SetOnHeal sets the function called when the entity is healed.
func (e *Living) SetOnHeal(f func(health float64, src world.HealingSource)) {
	e.onHeal = f
}

// Drops gets the drops of the entity.
func (e *Living) Drops() []item.Stack {
	return e.drops
}

// OnGround checks if the entity is considered to be on the ground.
func (e *Living) OnGround() bool {
	return e.onGround.Load()
}

// Dead checks if the entity is considered dead.
func (e *Living) Dead() bool {
	return e.health <= 0
}

// AttackImmune checks if the entity is currently immune to entity attacks
func (e *Living) AttackImmune() bool {
	return time.Since(e.lastAttack) <= 470*time.Millisecond
}

// Hurt hurts the entity for a given amount of damage.
func (e *Living) Hurt(damage float64, src world.DamageSource) (n float64, vulnerable bool) {
	if e.AttackImmune() {
		return 0, false
	}
	e.health -= damage
	if e.Dead() {
		for _, v := range e.viewers() {
			v.ViewEntityAction(e, entity.DeathAction{})
		}

		for _, drop := range e.drops {
			e.World().AddEntity(entity.NewItem(drop, e.pos))
		}

		time.AfterFunc(time.Second*2, func() {
			_ = e.Close()
			e.World().RemoveEntity(e)
		})
	}
	if s, ok := src.(entity.AttackDamageSource); ok {
		e.lastAttack = time.Now()
		e.KnockBack(s.Attacker.Position(), 0.4, 0.4)

		for _, v := range e.viewers() {
			v.ViewEntityAction(e, entity.HurtAction{})
		}
	}
	if e.onHurt != nil {
		damage, vulnerable = e.onHurt(damage, src)
	}
	return damage, vulnerable
}

// Heal heals the entity for a given amount of health
func (e *Living) Heal(health float64, src world.HealingSource) {
	e.health += health
	if e.health > e.maxHealth {
		e.health = e.maxHealth
	}
	e.onHeal(health, src)
}

// KnockBack knocks the entity back with a given force and height
func (e *Living) KnockBack(src mgl64.Vec3, force, height float64) {
	velocity := e.Position().Sub(src)
	velocity[1] = 0

	if velocity.Len() != 0 {
		velocity = velocity.Normalize().Mul(force)
	}
	velocity[1] = height

	e.SetVelocity(velocity.Mul(1))
}

// Velocity gets the entity's velocity.
func (e *Living) Velocity() mgl64.Vec3 {
	return e.vel
}

// SetVelocity updates the entity's velocity.
func (e *Living) SetVelocity(velocity mgl64.Vec3) {
	e.vel = velocity
}

// AddEffect ...
func (e *Living) AddEffect(effect.Effect) {
	return
}

// RemoveEffect ...
func (e *Living) RemoveEffect(effect.Type) {
	return
}

// Effects ...
func (e *Living) Effects() []effect.Effect {
	return []effect.Effect{}
}

// Speed ...
func (e *Living) Speed() float64 {
	return e.speed
}

// SetSpeed ...
func (e *Living) SetSpeed(f float64) {
	e.speed = f
}

// Close ...
func (e *Living) Close() error {
	return nil
}

// Position returns the position of the entity.
func (e *Living) Position() mgl64.Vec3 {
	return e.pos
}

// Rotation returns the rotation of the entity.
func (e *Living) Rotation() cube.Rotation {
	return e.rot
}

// World returns the world the entity is in.
func (e *Living) World() *world.World {
	return e.w
}

// Type returns the world.EntityType for the Entity.
func (e *Living) Type() world.EntityType {
	return e.entityType
}

// updateFallState is called to update the entities falling state.
func (e *Living) updateFallState(distanceThisTick float64) {
	fallDistance := e.fallDistance.Load()
	if e.OnGround() {
		if fallDistance > 0 {
			e.fall(fallDistance)
			e.ResetFallDistance()
		}
	} else if distanceThisTick < fallDistance {
		e.fallDistance.Sub(distanceThisTick)
	} else {
		e.ResetFallDistance()
	}
}

// fall is called when a falling entity hits the ground.
func (e *Living) fall(distance float64) {
	var (
		w   = e.World()
		pos = cube.PosFromVec3(e.Position())
		b   = w.Block(pos)
	)
	if len(b.Model().BBox(pos, w)) == 0 {
		pos = pos.Sub(cube.Pos{0, 1})
		b = w.Block(pos)
	}
	if h, ok := b.(block.EntityLander); ok {
		h.EntityLand(pos, w, e, &distance)
	}
	dmg := distance - 3
	if dmg < 0.5 {
		return
	}
	e.Hurt(math.Ceil(dmg), entity.FallDamageSource{})
}

// ResetFallDistance resets the player's fall distance.
func (e *Living) ResetFallDistance() {
	e.fallDistance.Store(0)
}

// FallDistance returns the player's fall distance.
func (e *Living) FallDistance() float64 {
	return e.fallDistance.Load()
}

// Tick ...
func (e *Living) Tick(w *world.World, current int64) {
	m := e.mc.TickMovement(e, e.Position(), e.Velocity(), cube.Rotation{e.rot.Yaw(), e.rot.Pitch()})
	m.Send()

	e.vel = m.Velocity()
	e.Move(m.Position().Sub(e.Position()), 0, 0)
}

// Move handles the entity's movement.
func (e *Living) Move(deltaPos mgl64.Vec3, deltaYaw, deltaPitch float64) {
	if e.Dead() || (deltaPos.ApproxEqual(mgl64.Vec3{}) && mgl64.FloatEqual(deltaYaw, 0) && mgl64.FloatEqual(deltaPitch, 0)) {
		return
	}

	var (
		w                     = e.World()
		pos                   = e.Position()
		yaw, pitch            = e.Rotation().Elem()
		res, resYaw, resPitch = pos.Add(deltaPos), yaw + deltaYaw, pitch + deltaPitch
	)

	for _, v := range e.viewers() {
		v.ViewEntityMovement(e, res, cube.Rotation{resYaw, resPitch}, e.OnGround())
	}

	e.pos = res
	e.rot = cube.Rotation{resYaw, resPitch}
	if deltaPos.Len() <= 3 {
		// Only update velocity if the player is not moving too fast to prevent potential OOMs.
		e.vel = deltaPos
		//e.checkBlockCollisions(deltaPos, w)
	}

	horizontalVel := deltaPos
	horizontalVel[1] = 0

	e.onGround.Store(e.checkOnGround(w))
	//e.updateFallState(deltaPos[1])
}

// checkOnGround checks if the entity is on the ground.
func (e *Living) checkOnGround(w *world.World) bool {
	box := e.Type().BBox(e).Translate(e.Position())

	b := box.Grow(1)

	min, max := cube.PosFromVec3(b.Min()), cube.PosFromVec3(b.Max())
	for x := min[0]; x <= max[0]; x++ {
		for z := min[2]; z <= max[2]; z++ {
			for y := min[1]; y < max[1]; y++ {
				pos := cube.Pos{x, y, z}
				boxList := w.Block(pos).Model().BBox(pos, w)
				for _, bb := range boxList {
					if bb.GrowVec3(mgl64.Vec3{0, 0.05}).Translate(pos.Vec3()).IntersectsWith(box) {
						return true
					}
				}
			}
		}
	}
	return false
}

// checkBlockCollisions checks the entity's block collisions.
func (e *Living) checkBlockCollisions(vel mgl64.Vec3, w *world.World) {
	entityBBox := e.Type().BBox(e).Translate(e.Position())
	deltaX, deltaY, deltaZ := vel[0], vel[1], vel[2]

	e.checkEntityInsiders(w, entityBBox)

	grown := entityBBox.Extend(vel).Grow(0.25)
	min, max := grown.Min(), grown.Max()
	minX, minY, minZ := int(math.Floor(min[0])), int(math.Floor(min[1])), int(math.Floor(min[2]))
	maxX, maxY, maxZ := int(math.Ceil(max[0])), int(math.Ceil(max[1])), int(math.Ceil(max[2]))

	// A prediction of one BBox per block, plus an additional 2, in case
	blocks := make([]cube.BBox, 0, (maxX-minX)*(maxY-minY)*(maxZ-minZ)+2)
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			for z := minZ; z <= maxZ; z++ {
				pos := cube.Pos{x, y, z}
				boxes := w.Block(pos).Model().BBox(pos, w)
				for _, box := range boxes {
					blocks = append(blocks, box.Translate(pos.Vec3()))
				}
			}
		}
	}

	// epsilon is the epsilon used for thresholds for change used for change in position and velocity.
	const epsilon = 0.001

	if !mgl64.FloatEqualThreshold(deltaY, 0, epsilon) {
		// First we move the entity BBox on the Y axis.
		for _, blockBBox := range blocks {
			deltaY = entityBBox.YOffset(blockBBox, deltaY)
		}
		entityBBox = entityBBox.Translate(mgl64.Vec3{0, deltaY})
	}
	if !mgl64.FloatEqualThreshold(deltaX, 0, epsilon) {
		// Then on the X axis.
		for _, blockBBox := range blocks {
			deltaX = entityBBox.XOffset(blockBBox, deltaX)
		}
		entityBBox = entityBBox.Translate(mgl64.Vec3{deltaX})
	}
	if !mgl64.FloatEqualThreshold(deltaZ, 0, epsilon) {
		// And finally on the Z axis.
		for _, blockBBox := range blocks {
			deltaZ = entityBBox.ZOffset(blockBBox, deltaZ)
		}
	}

	e.collidedHorizontally.Store(!mgl64.FloatEqual(deltaX, vel[0]) || !mgl64.FloatEqual(deltaZ, vel[2]))
	e.collidedVertically.Store(!mgl64.FloatEqual(deltaY, vel[1]))
}

// checkEntityInsiders checks if the entity is colliding with any EntityInsider blocks.
func (e *Living) checkEntityInsiders(w *world.World, entityBBox cube.BBox) {
	box := entityBBox.Grow(-0.0001)
	min, max := cube.PosFromVec3(box.Min()), cube.PosFromVec3(box.Max())

	for y := min[1]; y <= max[1]; y++ {
		for x := min[0]; x <= max[0]; x++ {
			for z := min[2]; z <= max[2]; z++ {
				blockPos := cube.Pos{x, y, z}
				b := w.Block(blockPos)
				if collide, ok := b.(block.EntityInsider); ok {
					collide.EntityInside(blockPos, w, e)
					if _, liquid := b.(world.Liquid); liquid {
						continue
					}
				}

				if l, ok := w.Liquid(blockPos); ok {
					if collide, ok := l.(block.EntityInsider); ok {
						collide.EntityInside(blockPos, w, e)
					}
				}
			}
		}
	}
}

// viewers returns a list of all viewers of the Player.
func (e *Living) viewers() []world.Viewer {
	return e.World().Viewers(e.Position())
}
