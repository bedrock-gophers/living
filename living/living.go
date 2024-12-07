package living

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"math"
	"time"
)

var _ = world.Entity(&Living{})

type Living struct {
	*entity.HealthManager

	handle *world.EntityHandle
	tx     *world.Tx
	data   *world.EntityData

	*livingData
}

func (p *Living) Hurt(dmg float64, src world.DamageSource) (float64, bool) {
	if p.Dead() || dmg < 0 {
		return 0, false
	}
	totalDamage := dmg
	damageLeft := totalDamage

	immune := time.Now().Before(p.immuneUntil)
	if immune {
		if damageLeft = damageLeft - p.lastDamage; damageLeft <= 0 {
			return 0, false
		}
		return 0, false
	}

	immunity := time.Second / 2
	//ctx := event.C(p)
	//if p.Handler().HandleHurt(ctx, &damageLeft, immune, &immunity, src); ctx.Cancelled() {
	//	return 0, false
	//}
	p.setAttackImmunity(immunity, totalDamage)
	p.AddHealth(-damageLeft)

	pos := p.Position()
	for _, viewer := range p.viewers() {
		viewer.ViewEntityAction(p, entity.HurtAction{})
	}
	if src.Fire() {
		p.tx.PlaySound(pos, sound.Burning{})
	} else if _, ok := src.(entity.DrowningDamageSource); ok {
		p.tx.PlaySound(pos, sound.Drowning{})
	}

	if p.Dead() {
		p.kill(src)
	}
	return totalDamage, true
}

func (p *Living) kill(src world.DamageSource) {
	for _, viewer := range p.viewers() {
		viewer.ViewEntityAction(p, entity.DeathAction{})
	}

	p.AddHealth(-p.MaxHealth())
	//p.Handler().HandleDeath(p, src, &keepInv)
	p.dropItems()

	// Wait a little before removing the entity. The client displays a death
	// animation while the player is dying.
	time.AfterFunc(time.Millisecond*1100, func() {
		p.H().ExecWorld(finishDying)
	})
}

// finishDying completes the death of a player, removing it from the world.
func finishDying(_ *world.Tx, e world.Entity) {
	p := e.(*Living)
	_ = p.Close()
}

func (p *Living) dropItems() {
	/*pos := p.Position()
	for _, orb := range entity.NewExperienceOrbs(pos, int(math.Min(float64(1*7), 100))) {
		p.tx.AddEntity(orb)
	}
	p.MoveItemsToInventory()
	for _, it := range append(p.inv.Clear(), append(p.armour.Clear(), p.offHand.Clear()...)...) {
		if _, ok := it.Enchantment(enchantment.CurseOfVanishing); ok {
			continue
		}
		opts := world.EntitySpawnOpts{Position: pos, Velocity: mgl64.Vec3{rand.Float64()*0.2 - 0.1, 0.2, rand.Float64()*0.2 - 0.1}}
		p.tx.AddEntity(entity.NewItem(opts, it))
	}*/
}

// setAttackImmunity sets the duration the player is immune to entity attacks.
func (p *Living) setAttackImmunity(d time.Duration, dmg float64) {
	p.immuneUntil = time.Now().Add(d)
	p.lastDamage = dmg
}

func (l Living) Heal(health float64, src world.HealingSource) {
	//TODO implement me
	panic("implement me")
}

// KnockBack knocks the player back with a given force and height. A source is passed which indicates the
// source of the velocity, typically the position of an attacking entity. The source is used to calculate the
// direction which the entity should be knocked back in.
func (p *Living) KnockBack(src mgl64.Vec3, force, height float64) {
	if p.Dead() {
		return
	}
	p.knockBack(src, force, height)
}

// knockBack is an unexported function that is used to knock the player back. This function does not check if the player
// can take damage or not.
func (p *Living) knockBack(src mgl64.Vec3, force, height float64) {
	velocity := p.Position().Sub(src)
	velocity[1] = 0

	if velocity.Len() != 0 {
		velocity = velocity.Normalize().Mul(force)
	}
	velocity[1] = height

	p.SetVelocity(velocity.Mul(1))
}

func (l Living) Velocity() mgl64.Vec3 {
	return l.data.Vel
}

func (p *Living) SetVelocity(velocity mgl64.Vec3) {
	p.data.Vel = velocity
	for _, v := range p.viewers() {
		v.ViewEntityVelocity(p, velocity)
	}
}

func (l Living) AddEffect(e effect.Effect) {
	//TODO implement me
	panic("implement me")
}

func (l Living) RemoveEffect(e effect.Type) {
	//TODO implement me
	panic("implement me")
}

func (l Living) Effects() []effect.Effect {
	return nil
}

func (l Living) Speed() float64 {
	return l.speed
}

func (l Living) SetSpeed(f float64) {

}

func (l Living) Close() error {
	l.tx.RemoveEntity(l)
	return nil
}

func (l Living) H() *world.EntityHandle {
	return l.handle
}

func (l Living) Position() mgl64.Vec3 {
	return l.data.Pos
}

func (l Living) Rotation() cube.Rotation {
	return l.data.Rot
}

func (l *Living) Dead() bool {
	return l.Health() <= mgl64.Epsilon
}

func (l *Living) OnGround() bool {
	return l.onGround
}

func (l *Living) EyeHeight() float64 {
	return 1.62
}

// Move moves the player from one position to another in the world, by adding the delta passed to the current
// position of the player.
// Move also rotates the player, adding deltaYaw and deltaPitch to the respective values.
func (p *Living) Move(deltaPos mgl64.Vec3, deltaYaw, deltaPitch float64) {
	if p.Dead() || (deltaPos.ApproxEqual(mgl64.Vec3{}) && mgl64.FloatEqual(deltaYaw, 0) && mgl64.FloatEqual(deltaPitch, 0)) {
		return
	}
	if p.immobile {
		if mgl64.FloatEqual(deltaYaw, 0) && mgl64.FloatEqual(deltaPitch, 0) {
			// If only the position was changed, don't continue with the movement when immobile.
			return
		}
		// Still update rotation if it was changed.
		deltaPos = mgl64.Vec3{}
	}
	var (
		pos         = p.Position()
		yaw, pitch  = p.Rotation().Elem()
		res, resRot = pos.Add(deltaPos), cube.Rotation{yaw + deltaYaw, pitch + deltaPitch}
	)

	for _, v := range p.viewers() {
		v.ViewEntityMovement(p, res, resRot, p.OnGround())
	}

	p.data.Pos = res
	p.data.Rot = resRot
	if deltaPos.Len() <= 3 {
		// Only update velocity if the player is not moving too fast to prevent potential OOMs.
		p.data.Vel = deltaPos
		p.checkBlockCollisions(deltaPos)
	}

	horizontalVel := deltaPos
	horizontalVel[1] = 0

	p.onGround = p.checkOnGround()
	p.updateFallState(deltaPos[1])
}

// ResetFallDistance resets the player's fall distance.
func (p *Living) ResetFallDistance() {
	p.fallDistance = 0
}

// FallDistance returns the player's fall distance.
func (p *Living) FallDistance() float64 {
	return p.fallDistance
}

// OnFireDuration ...
func (p *Living) OnFireDuration() time.Duration {
	return time.Duration(p.fireTicks) * time.Second / 20
}

// SetOnFire ...
func (p *Living) SetOnFire(duration time.Duration) {
	ticks := int64(duration.Seconds() * 20)
	p.fireTicks = ticks
	p.updateState()
}

// Extinguish ...
func (p *Living) Extinguish() {
	p.SetOnFire(0)
}

// Tick ticks the entity, performing actions such as checking if the player is still breaking a block.
func (l *Living) Tick(_ *world.Tx, current int64) {
	if l.Dead() {
		return
	}

	l.checkBlockCollisions(l.data.Vel)
	l.onGround = l.checkOnGround()

	if l.Position()[1] < float64(l.tx.Range()[0]) && current%10 == 0 {
		l.Hurt(4, entity.VoidDamageSource{})
	}
	if l.insideOfSolid() {
		l.Hurt(1, entity.SuffocationDamageSource{})
	}

	if l.OnFireDuration() > 0 {
		l.fireTicks -= 1
		if l.OnFireDuration() <= 0 || l.tx.RainingAt(cube.PosFromVec3(l.Position())) {
			l.Extinguish()
		}
		if l.OnFireDuration()%time.Second == 0 {
			l.Hurt(1, block.FireDamageSource{})
		}
	}

	m := l.mc.TickMovement(l, l.Position(), l.Velocity(), l.Rotation(), l.tx)
	m.Send()

	l.data.Vel = m.Velocity()
	l.Move(m.Position().Sub(l.Position()), 0, 0)
}

// updateFallState is called to update the entities falling state.
func (p *Living) updateFallState(distanceThisTick float64) {
	if p.OnGround() {
		if p.fallDistance > 0 {
			p.fall(p.fallDistance)
			p.ResetFallDistance()
		}
	} else if distanceThisTick < p.fallDistance {
		p.fallDistance -= distanceThisTick
	} else {
		p.ResetFallDistance()
	}
}

// fall is called when a falling entity hits the ground.
func (p *Living) fall(distance float64) {
	pos := cube.PosFromVec3(p.Position())
	b := p.tx.Block(pos)

	if len(b.Model().BBox(pos, p.tx)) == 0 {
		pos = pos.Sub(cube.Pos{0, 1})
		b = p.tx.Block(pos)
	}
	if h, ok := b.(block.EntityLander); ok {
		h.EntityLand(pos, p.tx, p, &distance)
	}
	dmg := distance - 3
	if dmg < 0.5 {
		return
	}
	p.Hurt(math.Ceil(dmg), entity.FallDamageSource{})
}

// checkCollisions checks the player's block collisions.
func (l *Living) checkBlockCollisions(vel mgl64.Vec3) {
	entityBBox := l.entityType.BBox(l).Translate(l.Position())
	deltaX, deltaY, deltaZ := vel[0], vel[1], vel[2]

	l.checkEntityInsiders(entityBBox)

	grown := entityBBox.Extend(vel).Grow(0.25)
	low, high := grown.Min(), grown.Max()
	minX, minY, minZ := int(math.Floor(low[0])), int(math.Floor(low[1])), int(math.Floor(low[2]))
	maxX, maxY, maxZ := int(math.Ceil(high[0])), int(math.Ceil(high[1])), int(math.Ceil(high[2]))

	// A prediction of one BBox per block, plus an additional 2, in case
	blocks := make([]cube.BBox, 0, (maxX-minX)*(maxY-minY)*(maxZ-minZ)+2)
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			for z := minZ; z <= maxZ; z++ {
				pos := cube.Pos{x, y, z}
				boxes := l.tx.Block(pos).Model().BBox(pos, l.tx)
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

	l.collidedHorizontally = !mgl64.FloatEqual(deltaX, vel[0]) || !mgl64.FloatEqual(deltaZ, vel[2])
	l.collidedVertically = !mgl64.FloatEqual(deltaY, vel[1])
}

// checkEntityInsiders checks if the player is colliding with any EntityInsider blocks.
func (l *Living) checkEntityInsiders(entityBBox cube.BBox) {
	box := entityBBox.Grow(-0.0001)
	low, high := cube.PosFromVec3(box.Min()), cube.PosFromVec3(box.Max())

	for y := low[1]; y <= high[1]; y++ {
		for x := low[0]; x <= high[0]; x++ {
			for z := low[2]; z <= high[2]; z++ {
				blockPos := cube.Pos{x, y, z}
				b := l.tx.Block(blockPos)
				if collide, ok := b.(block.EntityInsider); ok {
					collide.EntityInside(blockPos, l.tx, l)
					if _, liquid := b.(world.Liquid); liquid {
						continue
					}
				}

				if lq, ok := l.tx.Liquid(blockPos); ok {
					if collide, ok := lq.(block.EntityInsider); ok {
						collide.EntityInside(blockPos, l.tx, l)
					}
				}
			}
		}
	}
}

// checkOnGround checks if the player is currently considered to be on the ground.
func (l *Living) checkOnGround() bool {
	box := l.entityType.BBox(l).Translate(l.Position())
	b := box.Grow(1)

	low, high := cube.PosFromVec3(b.Min()), cube.PosFromVec3(b.Max())
	for x := low[0]; x <= high[0]; x++ {
		for z := low[2]; z <= high[2]; z++ {
			for y := low[1]; y < high[1]; y++ {
				pos := cube.Pos{x, y, z}
				boxList := l.tx.Block(pos).Model().BBox(pos, l.tx)
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

func (l *Living) viewers() []world.Viewer {
	return l.tx.Viewers(l.data.Pos)
}

// insideOfSolid returns true if the player is inside a solid block.
func (p *Living) insideOfSolid() bool {
	pos := cube.PosFromVec3(entity.EyePosition(p))
	b, box := p.tx.Block(pos), p.handle.Type().BBox(p).Translate(p.Position())

	_, solid := b.Model().(model.Solid)
	if !solid {
		// Not solid.
		return false
	}
	d, diffuses := b.(block.LightDiffuser)
	if diffuses && d.LightDiffusionLevel() == 0 {
		// Transparent.
		return false
	}
	for _, blockBox := range b.Model().BBox(pos, p.tx) {
		if blockBox.Translate(pos.Vec3()).IntersectsWith(box) {
			return true
		}
	}
	return false
}

// updateState updates the state of the player to all viewers of the player.
func (p *Living) updateState() {
	for _, v := range p.viewers() {
		v.ViewEntityState(p)
	}
}
