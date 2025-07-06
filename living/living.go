package living

import (
	"iter"
	"maps"
	"math"
	"slices"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/item/enchantment"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

var _ = world.Entity(&Living{})
var _ = entity.Living(&Living{})

type Living struct {
	handle *world.EntityHandle
	tx     *world.Tx
	data   *world.EntityData

	*livingData
}

func (l *Living) Heal(health float64, _ world.HealingSource) {
	l.AddHealth(health)
}

func (l *Living) Hurt(dmg float64, src world.DamageSource) (float64, bool) {
	if l.Dead() || dmg <= 0 {
		return 0, false
	}
	totalDamage := dmg
	damageLeft := totalDamage

	immune := time.Now().Before(l.immuneUntil)
	if immune {
		if damageLeft = damageLeft - l.lastDamage; damageLeft <= 0 {
			return 0, false
		}
	}

	immunity := l.immuneDuration
	ctx := event.C[*Living](l)
	if l.handler.HandleHurt(*ctx, totalDamage, immune, &immunity, src); ctx.Cancelled() {
		return 0, false
	}
	l.setAttackImmunity(immunity, totalDamage)
	l.AddHealth(-damageLeft)

	pos := l.Position()
	for _, viewer := range l.Viewers(l.tx) {
		viewer.ViewEntityAction(l, entity.HurtAction{})
	}
	if src.Fire() {
		l.tx.PlaySound(pos, sound.Burning{})
	} else if _, ok := src.(entity.DrowningDamageSource); ok {
		l.tx.PlaySound(pos, sound.Drowning{})
	}

	if l.Dead() {
		l.Kill(src)
	}
	return totalDamage, true
}

func (l *Living) Kill(_ world.DamageSource) {
	for _, viewer := range l.Viewers(l.tx) {
		viewer.ViewEntityAction(l, entity.DeathAction{})
	}

	l.AddHealth(-l.MaxHealth())
	l.DropItems(l.tx)

	// Wait a little before removing the entity. The client displays a death
	// animation while the player is dying.
	time.AfterFunc(time.Millisecond*1100, func() {
		l.H().ExecWorld(finishDying)
	})
}

// finishDying completes the death of a player, removing it from the world.
func finishDying(_ *world.Tx, e world.Entity) {
	p := e.(entity.Living)
	_ = p.Close()
}

func (l *Living) DropItems(tx *world.Tx) {
	pos := l.Position()
	for d := range l.drops {
		it := d.Stack()
		if it.Empty() {
			continue
		}
		if _, ok := it.Enchantment(enchantment.CurseOfVanishing); ok {
			continue
		}
		opts := world.EntitySpawnOpts{Position: pos}
		tx.AddEntity(entity.NewItem(opts, it))
	}
}

// setAttackImmunity sets the duration the player is immune to entity attacks.
func (l *Living) setAttackImmunity(d time.Duration, dmg float64) {
	l.immuneUntil = time.Now().Add(d)
	l.lastDamage = dmg
}

// KnockBack knocks the player back with a given force and height. A source is passed which indicates the
// source of the velocity, typically the position of an attacking entity. The source is used to calculate the
// direction which the entity should be knocked back in.
func (l *Living) KnockBack(src mgl64.Vec3, force, height float64) {
	if l.Dead() {
		return
	}
	l.knockBack(src, force, height)
}

// knockBack is an unexported function that is used to knock the player back. This function does not check if the player
// can take damage or not.
func (l *Living) knockBack(src mgl64.Vec3, force, height float64) {
	velocity := l.Position().Sub(src)
	velocity[1] = 0

	if velocity.Len() != 0 {
		velocity = velocity.Normalize().Mul(force)
	}
	velocity[1] = height

	l.SetVelocity(velocity.Mul(1))
}

// Tx returns the transaction.
func (l *Living) Tx() *world.Tx {
	return l.tx
}

// Age returns the age of the entity.
func (l *Living) Age() time.Duration {
	return l.age
}

// Speed returns the speed.
func (l *Living) Speed() float64 {
	return l.speed
}

// SetSpeed sets the speed.
func (l *Living) SetSpeed(f float64) {
	l.speed = f
}

// Velocity returns the velocity.
func (l *Living) Velocity() mgl64.Vec3 {
	return l.data.Vel
}

// SetVelocity sets the velocity.
func (l *Living) SetVelocity(velocity mgl64.Vec3) {
	l.data.Vel = velocity
	for _, v := range l.Viewers(l.tx) {
		v.ViewEntityVelocity(l, velocity)
	}
}

// Drops returns the drops.
func (l *Living) Drops() iter.Seq[Drop] {
	return l.drops
}

// AddEffect adds an effect to the entity.
func (l *Living) AddEffect(e effect.Effect) {
	l.effects[e.Type()] = e
}

// RemoveEffect removes the effect of an entity.
func (l *Living) RemoveEffect(e effect.Type) {
	delete(l.effects, e)
}

// Effects returns the effects of an entity.
func (l *Living) Effects() []effect.Effect {
	return slices.Collect(maps.Values(l.effects))
}

// Close closes the entity.
func (l *Living) Close() error {
	l.tx.RemoveEntity(l)
	return nil
}

// H returns the EntityHandle.
func (l *Living) H() *world.EntityHandle {
	return l.handle
}

// Position returns the position.
func (l *Living) Position() mgl64.Vec3 {
	return l.data.Pos
}

// Rotation returns the rotation.
func (l *Living) Rotation() cube.Rotation {
	return l.data.Rot
}

// Dead returns if the entity is dead or not.
func (l *Living) Dead() bool {
	return l.Health() <= mgl64.Epsilon
}

// OnGround returns if the entity is on the ground.
func (l *Living) OnGround() bool {
	return l.onGround
}

// Immobile returns if the entity is Immobile.
func (l *Living) Immobile() bool {
	return l.immobile
}

// SetImmobile sets if the entity is immobile or not.
func (l *Living) SetImmobile(immobile bool, tx *world.Tx) {
	l.immobile = immobile
	for _, v := range l.Viewers(tx) {
		v.ViewEntityState(l)
	}
}

// Invisible ...
func (l *Living) Invisible() bool {
	return l.invisible
}

// SetInvisible ...
func (l *Living) SetInvisible(invisible bool, tx *world.Tx) {
	l.invisible = invisible
	for _, v := range l.Viewers(tx) {
		v.ViewEntityState(l)
	}
}

// Scale ...
func (l *Living) Scale() float64 {
	return l.scale
}

// SetScale ...
func (l *Living) SetScale(scale float64, tx *world.Tx) {
	l.scale = scale
	for _, v := range l.Viewers(tx) {
		v.ViewEntityState(l)
	}
}

// EyeHeight ...
func (l *Living) EyeHeight() float64 {
	return l.livingData.eyeHeight
}

// NameTag ...
func (l *Living) NameTag() string {
	return l.data.Name
}

// SetNameTag ...
func (l *Living) SetNameTag(s string, tx *world.Tx) {
	l.data.Name = s
	for _, v := range l.Viewers(tx) {
		v.ViewEntityState(l)
	}
}

// Move moves the player from one position to another in the world, by adding the delta passed to the current
// position of the player.
// Move also rotates the player, adding deltaYaw and deltaPitch to the respective values.
func (l *Living) Move(deltaPos mgl64.Vec3, deltaYaw, deltaPitch float64, tx *world.Tx) {
	if l.Dead() || (deltaPos.ApproxEqual(mgl64.Vec3{}) && mgl64.FloatEqual(deltaYaw, 0) && mgl64.FloatEqual(deltaPitch, 0)) {
		return
	}
	if l.immobile {
		if mgl64.FloatEqual(deltaYaw, 0) && mgl64.FloatEqual(deltaPitch, 0) {
			// If only the position was changed, don't continue with the movement when Immobile.
			return
		}
		// Still update rotation if it was changed.
		deltaPos = mgl64.Vec3{}
	}
	var (
		pos         = l.Position()
		yaw, pitch  = l.Rotation().Elem()
		res, resRot = pos.Add(deltaPos), cube.Rotation{yaw + deltaYaw, pitch + deltaPitch}
	)

	for _, v := range l.Viewers(tx) {
		v.ViewEntityMovement(l, res, resRot, l.OnGround())
	}

	l.data.Pos = res
	l.data.Rot = resRot
	if deltaPos.Len() <= 3 {
		// Only update velocity if the player is not moving too fast to prevent potential OOMs.
		l.data.Vel = deltaPos
		l.checkBlockCollisions(deltaPos, tx)
	}

	l.onGround = l.checkOnGround(tx)
	//l.updateFallState(deltaPos[1], tx)
}

// MoveToTarget Target is assumed to be another Entity or similar struct with position getters.
func (l *Living) MoveToTarget(target mgl64.Vec3, jumpVelocity float64, tx *world.Tx) {
	if l.Dead() {
		return
	}

	delta := target.Sub(l.Position())
	delta[1] = 0
	if delta.Len() == 0 {
		return
	}
	dir := delta.Normalize()
	baseMove := dir.Mul(l.Speed())

	checkOffset := dir.Mul(l.H().Type().BBox(l).Width())
	checkPos := cube.PosFromVec3(l.Position().Add(checkOffset))
	low := tx.Block(checkPos)
	high := tx.Block(checkPos.Add(cube.Pos{0, 1, 0}))

	_, solidLow := low.Model().(model.Solid)
	_, solidHigh := high.Model().(model.Solid)

	move := baseMove

	if solidLow {
		maxY := 0.0
		for _, box := range low.Model().BBox(cube.Pos{}, tx) {
			if h := box.Max()[1]; h > maxY {
				maxY = h
			}
		}

		if !solidHigh {
			move[1] = jumpVelocity
			if l.OnGround() {
				move[0] *= 0.30
				move[2] *= 0.30
			}
		} else {
			move[0], move[2] = 0, 0
		}
	}

	if !l.OnGround() && move[1] == 0 {
		move[0] *= 0.25
		move[2] *= 0.25
	}

	l.Move(move, 0, 0, tx)
}

// LookAt ...
func (l *Living) LookAt(v mgl64.Vec3, tx *world.Tx) {
	yaw, pitch := LookAtExtended(l.Position().Add(mgl64.Vec3{0, l.EyeHeight(), 0}), v)
	dy := yaw - l.Rotation().Yaw()
	dp := pitch - l.Rotation().Pitch()

	l.Move(mgl64.Vec3{0, 0, 0}, dy, dp, tx)
}

// LookAwayFrom ...
func (l *Living) LookAwayFrom(v mgl64.Vec3, tx *world.Tx) {
	yaw, pitch := LookAtExtended(l.Position().Add(mgl64.Vec3{0, l.EyeHeight(), 0}), v)
	dy := int(math.Round(yaw - l.Rotation().Yaw()))
	dp := pitch - l.Rotation().Pitch()

	dy = (dy + 180) % 360
	if dy > 180 {
		dy -= 360
	}

	l.Move(mgl64.Vec3{0, 0, 0}, float64(dy), -dp, tx)
}

// LookAtExtended ...
func LookAtExtended(pos mgl64.Vec3, v mgl64.Vec3) (yaw float64, pitch float64) {
	vt := v.Y() - pos.Y()
	hz := math.Sqrt(math.Pow(v.X()-pos.X(), 2) + math.Pow(v.Z()-pos.Z(), 2))
	pitch = (-math.Atan2(vt, hz) / math.Pi) * 180

	dz := v.Z() - pos.Z()
	dx := v.X() - pos.X()
	yaw = (math.Atan2(dz, dx)/math.Pi)*180 - 90
	if yaw < 0 {
		yaw += 360.0
	}

	return yaw, pitch
}

// ResetFallDistance resets the player's fall distance.
func (l *Living) ResetFallDistance() {
	l.fallDistance = 0
}

// FallDistance returns the player's fall distance.
func (l *Living) FallDistance() float64 {
	return l.fallDistance
}

// OnFireDuration ...
func (l *Living) OnFireDuration() time.Duration {
	return time.Duration(l.fireTicks) * time.Second / 20
}

// SetOnFire ...
func (l *Living) SetOnFire(duration time.Duration) {
	ticks := int64(duration.Seconds() * 20)
	l.fireTicks = ticks
	l.updateState()
}

// Extinguish ...
func (l *Living) Extinguish() {
	l.SetOnFire(0)
}

// ImmuneUntil ...
func (l *Living) ImmuneUntil() time.Time {
	return l.immuneUntil
}

// SetImmuneDuration ...
func (l *Living) SetImmuneDuration(duration time.Duration) {
	l.immuneDuration = duration
}

// ImmuneDuration ...
func (l *Living) ImmuneDuration() time.Duration {
	return l.immuneDuration
}

// AttackImmune ...
func (l *Living) AttackImmune() bool {
	return l.ImmuneUntil().After(time.Now())
}

// LastDamage ...
func (l *Living) LastDamage() float64 {
	return l.lastDamage
}

// Tick ticks the entity, performing actions such as checking if the player is still breaking a block.
func (l *Living) Tick(tx *world.Tx, current int64) {
	l.age += 50 * time.Millisecond
	ctx := event.C(l)
	l.handler.HandleTick(*ctx, tx)

	if ctx.Cancelled() || l.Dead() {
		return
	}

	l.checkBlockCollisions(l.data.Vel, tx)
	l.onGround = l.checkOnGround(tx)

	if l.Position()[1] < float64(tx.Range()[0]) && current%10 == 0 {
		l.Hurt(4, entity.VoidDamageSource{})
	}

	if l.OnFireDuration() > 0 {
		l.fireTicks -= 1
		if l.OnFireDuration() <= 0 || tx.RainingAt(cube.PosFromVec3(l.Position())) {
			l.Extinguish()
		}
		if l.OnFireDuration()%time.Second == 0 {
			l.Hurt(1, block.FireDamageSource{})
		}
	}

	m := l.mc.TickMovement(l, l.Position(), l.Velocity(), l.Rotation(), tx)
	m.Send()

	l.data.Vel = m.Velocity()
	l.Move(m.Position().Sub(l.Position()), 0, 0, tx)
}

// Variant ...
func (l *Living) Variant() int32 {
	return l.variant
}

// WithVariant ...
func (l *Living) WithVariant(v int32) {
	l.variant = v
}

// MarkVariant ...
func (l *Living) MarkVariant() int32 {
	return l.markVariant
}

// WithMarkVariant ...
func (l *Living) WithMarkVariant(v int32, tx *world.Tx) {
	l.markVariant = v
	for _, v := range l.Viewers(tx) {
		v.ViewEntityState(l)
	}
}

// updateFallState is called to update the entities falling state.
func (l *Living) updateFallState(distanceThisTick float64, tx *world.Tx) {
	if l.OnGround() {
		if l.fallDistance > 0 {
			l.fall(l.fallDistance, tx)
			l.ResetFallDistance()
		}
	} else if distanceThisTick < l.fallDistance {
		l.fallDistance -= distanceThisTick
	} else {
		l.ResetFallDistance()
	}
}

// fall is called when a falling entity hits the ground.
func (l *Living) fall(distance float64, tx *world.Tx) {
	pos := cube.PosFromVec3(l.Position())
	b := tx.Block(pos)

	if len(b.Model().BBox(pos, tx)) == 0 {
		pos = pos.Sub(cube.Pos{0, 1})
		b = tx.Block(pos)
	}
	if h, ok := b.(block.EntityLander); ok {
		h.EntityLand(pos, tx, l, &distance)
	}
	dmg := distance - 3
	if dmg < 0.5 {
		return
	}
	l.Hurt(math.Ceil(dmg), entity.FallDamageSource{})
}

// checkCollisions checks the player's block collisions.
func (l *Living) checkBlockCollisions(vel mgl64.Vec3, tx *world.Tx) {
	entityBBox := l.entityType.BBox(l).Translate(l.Position())
	deltaX, deltaY, deltaZ := vel[0], vel[1], vel[2]

	l.checkEntityInsiders(entityBBox, tx)

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
				boxes := tx.Block(pos).Model().BBox(pos, tx)
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
func (l *Living) checkEntityInsiders(entityBBox cube.BBox, tx *world.Tx) {
	box := entityBBox.Grow(-0.0001)
	low, high := cube.PosFromVec3(box.Min()), cube.PosFromVec3(box.Max())

	for y := low[1]; y <= high[1]; y++ {
		for x := low[0]; x <= high[0]; x++ {
			for z := low[2]; z <= high[2]; z++ {
				blockPos := cube.Pos{x, y, z}
				b := tx.Block(blockPos)
				if collide, ok := b.(block.EntityInsider); ok {
					collide.EntityInside(blockPos, tx, l)
					if _, liquid := b.(world.Liquid); liquid {
						continue
					}
				}

				if lq, ok := tx.Liquid(blockPos); ok {
					if collide, ok := lq.(block.EntityInsider); ok {
						collide.EntityInside(blockPos, tx, l)
					}
				}
			}
		}
	}
}

// checkOnGround checks if the player is currently considered to be on the ground.
func (l *Living) checkOnGround(tx *world.Tx) bool {
	box := l.entityType.BBox(l).Translate(l.Position())
	b := box.Grow(1)

	low, high := cube.PosFromVec3(b.Min()), cube.PosFromVec3(b.Max())
	for x := low[0]; x <= high[0]; x++ {
		for z := low[2]; z <= high[2]; z++ {
			for y := low[1]; y < high[1]; y++ {
				pos := cube.Pos{x, y, z}
				boxList := tx.Block(pos).Model().BBox(pos, tx)
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

// Viewers returns the viewers.
func (l *Living) Viewers(tx *world.Tx) []world.Viewer {
	return tx.Viewers(l.data.Pos)
}

// insideOfSolid returns true if the player is inside a solid block.
func (l *Living) insideOfSolid(tx *world.Tx) bool {
	pos := cube.PosFromVec3(entity.EyePosition(l))
	b, box := tx.Block(pos), l.handle.Type().BBox(l).Translate(l.Position())

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
	for _, blockBox := range b.Model().BBox(pos, tx) {
		if blockBox.Translate(pos.Vec3()).IntersectsWith(box) {
			return true
		}
	}
	return false
}

// updateState updates the state of the player to all Viewers of the player.
func (l *Living) updateState() {
	for _, v := range l.Viewers(l.tx) {
		v.ViewEntityState(l)
	}
}
