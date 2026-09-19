// Package npc provides player-shaped, server-controlled Living entities.
package npc

import (
	"errors"
	"time"

	"github.com/bedrock-gophers/living/living"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/enchantment"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/df-mc/dragonfly/server/player/dialogue"
	"github.com/df-mc/dragonfly/server/player/form"
	"github.com/df-mc/dragonfly/server/player/hud"
	"github.com/df-mc/dragonfly/server/player/input"
	"github.com/df-mc/dragonfly/server/player/skin"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"golang.org/x/text/language"
)

var _ session.Controllable = (*Player)(nil)

// Handler handles NPC-specific events.
type Handler interface {
	HandleAttackEntity(npc *Player, target world.Entity, force, height *float64, critical *bool) bool
	HandleHurt(npc *Player, damage *float64, immune bool, immunity *time.Duration, source world.DamageSource) bool
}

// NopHandler implements Handler without cancelling events.
type NopHandler struct{}

func (NopHandler) HandleAttackEntity(*Player, world.Entity, *float64, *float64, *bool) bool {
	return false
}
func (NopHandler) HandleHurt(*Player, *float64, bool, *time.Duration, world.DamageSource) bool {
	return false
}

// Config configures a player-shaped Living entity.
type Config struct {
	Living living.Config
	Name   string
	UUID   uuid.UUID
	Skin   skin.Skin

	Inventory  *inventory.Inventory
	OffHand    *inventory.Inventory
	Armour     *inventory.Armour
	EnderChest *inventory.Inventory
	Handler    Handler
}

type data struct {
	name                                                                string
	id                                                                  uuid.UUID
	skin                                                                skin.Skin
	inventory, offHand, enderChest                                      *inventory.Inventory
	armour                                                              *inventory.Armour
	handler                                                             Handler
	heldSlot                                                            int
	gameMode                                                            world.GameMode
	food, experienceLevel                                               int
	experienceProgress                                                  float64
	sprinting, sneaking, swimming, crawling, flying, gliding, usingItem bool
	cooldowns                                                           map[string]time.Time
}

func (c Config) Apply(entityData *world.EntityData) {
	if c.UUID == uuid.Nil {
		c.UUID = uuid.New()
	}
	if c.Inventory == nil {
		c.Inventory = inventory.New(36, nil)
	}
	if c.OffHand == nil {
		c.OffHand = inventory.New(1, nil)
	}
	if c.Armour == nil {
		c.Armour = inventory.NewArmour(nil)
	}
	if c.EnderChest == nil {
		c.EnderChest = inventory.New(27, nil)
	}
	if c.Handler == nil {
		c.Handler = NopHandler{}
	}
	state := &data{name: c.Name, id: c.UUID, skin: c.Skin, inventory: c.Inventory, offHand: c.OffHand,
		armour: c.Armour, enderChest: c.EnderChest, handler: c.Handler, gameMode: world.GameModeSurvival,
		food: 20, cooldowns: make(map[string]time.Time)}
	c.Living.EntityType = Type
	c.Living.Attachment = state
	c.Living.Handler = livingHandler{state: state}
	c.Living.Apply(entityData)
}

type livingHandler struct{ state *data }

func (livingHandler) HandleTick(living.Context, *world.Tx) {}
func (h livingHandler) HandleHurt(ctx living.Context, damage float64, immune bool, immunity *time.Duration, source world.DamageSource) {
	h.HandleMutableHurt(ctx, &damage, immune, immunity, source)
}
func (h livingHandler) HandleMutableHurt(ctx living.Context, damage *float64, immune bool, immunity *time.Duration, source world.DamageSource) {
	base := ctx.Val()
	entity, ok := base.H().Entity(base.Tx())
	if ok && h.state.handler.HandleHurt(entity.(*Player), damage, immune, immunity, source) {
		ctx.Cancel()
	}
}

type npcType struct{ living.NopLivingType }

// Type is the entity type used by player-shaped NPCs.
var Type npcType

func (npcType) EncodeEntity() string        { return "minecraft:player" }
func (npcType) BBox(world.Entity) cube.BBox { return cube.Box(-0.3, 0, -0.3, 0.3, 1.8, 0.3) }
func (npcType) Open(tx *world.Tx, handle *world.EntityHandle, entityData *world.EntityData) world.Entity {
	base := living.Open(tx, handle, entityData)
	return &Player{Living: base, data: base.Attachment().(*data)}
}

// Player is a server-controlled player-shaped Living entity.
type Player struct {
	*living.Living
	data *data
}

// SetHandler replaces the NPC event handler.
func (p *Player) SetHandler(handler Handler) {
	if handler == nil {
		handler = NopHandler{}
	}
	p.data.handler = handler
}

// Handler returns the NPC event handler.
func (p *Player) Handler() Handler { return p.data.handler }

// HandleEnderPearlRefund forwards an optional failed-pearl callback to the handler.
func (p *Player) HandleEnderPearlRefund() {
	if handler, ok := p.data.handler.(interface{ HandleEnderPearlRefund(*Player) }); ok {
		handler.HandleEnderPearlRefund(p)
	}
}

func (p *Player) Name() string    { return p.data.name }
func (p *Player) UUID() uuid.UUID { return p.data.id }
func (*Player) XUID() string      { return "" }
func (p *Player) Skin() skin.Skin { return p.data.skin }
func (p *Player) SetSkin(value skin.Skin) {
	p.data.skin = value
	for _, viewer := range p.Viewers() {
		viewer.ViewSkin(p)
	}
}
func (*Player) Locale() language.Tag                        { return language.AmericanEnglish }
func (p *Player) Inventory() *inventory.Inventory           { return p.data.inventory }
func (p *Player) Armour() *inventory.Armour                 { return p.data.armour }
func (p *Player) EnderChestInventory() *inventory.Inventory { return p.data.enderChest }
func (p *Player) HeldItems() (item.Stack, item.Stack) {
	main, _ := p.data.inventory.Item(p.data.heldSlot)
	off, _ := p.data.offHand.Item(0)
	return main, off
}
func (p *Player) SetHeldItems(main, off item.Stack) {
	_ = p.data.inventory.SetItem(p.data.heldSlot, main)
	_ = p.data.offHand.SetItem(0, off)
	for _, viewer := range p.Viewers() {
		viewer.ViewEntityItems(p)
	}
}
func (p *Player) SetHeldSlot(slot int) error {
	if slot < 0 || slot > 8 {
		return errors.New("held slot must be in range 0-8")
	}
	p.data.heldSlot = slot
	for _, viewer := range p.Viewers() {
		viewer.ViewEntityItems(p)
	}
	return nil
}

// RefreshEquipment sends the NPC's held items and armour to current viewers.
func (p *Player) RefreshEquipment() {
	for _, viewer := range p.Viewers() {
		viewer.ViewEntityItems(p)
		viewer.ViewEntityArmour(p)
	}
}

// Hurt applies armour and resistance before delegating health and immunity handling to Living.
func (p *Player) Hurt(damage float64, source world.DamageSource) (float64, bool) {
	if _, ok := p.Effect(effect.FireResistance); (ok && source.Fire()) || p.Dead() || !p.GameMode().AllowsTakingDamage() || damage < 0 {
		return 0, false
	}
	damage = max(damage, 0)
	damage -= p.Armour().DamageReduction(damage, source)
	if resistance, ok := p.Effect(effect.Resistance); ok {
		damage *= effect.Resistance.Multiplier(source, resistance.Level())
	}
	return p.Living.Hurt(damage, source)
}

func (p *Player) GameMode() world.GameMode        { return p.data.gameMode }
func (p *Player) SetGameMode(mode world.GameMode) { p.data.gameMode = mode }
func (p *Player) Food() int                       { return p.data.food }
func (p *Player) SetFood(food int)                { p.data.food = max(0, min(20, food)) }
func (*Player) Absorption() float64               { return 0 }
func (p *Player) ExperienceLevel() int            { return p.data.experienceLevel }
func (p *Player) ExperienceProgress() float64     { return p.data.experienceProgress }
func (p *Player) SetExperienceLevel(level int)    { p.data.experienceLevel = max(0, level) }
func (*Player) EnchantmentSeed() int64            { return 0 }
func (*Player) ResetEnchantmentSeed()             {}

func (p *Player) StartSprinting()            { p.data.sprinting = true }
func (p *Player) Sprinting() bool            { return p.data.sprinting }
func (p *Player) StopSprinting()             { p.data.sprinting = false }
func (p *Player) StartSneaking()             { p.data.sneaking = true }
func (p *Player) Sneaking() bool             { return p.data.sneaking }
func (p *Player) StopSneaking()              { p.data.sneaking = false }
func (p *Player) StartSwimming()             { p.data.swimming = true }
func (p *Player) Swimming() bool             { return p.data.swimming }
func (p *Player) StopSwimming()              { p.data.swimming = false }
func (p *Player) StartCrawling()             { p.data.crawling = true }
func (p *Player) Crawling() bool             { return p.data.crawling }
func (p *Player) StopCrawling()              { p.data.crawling = false }
func (p *Player) StartFlying()               { p.data.flying = true }
func (p *Player) Flying() bool               { return p.data.flying }
func (p *Player) StopFlying()                { p.data.flying = false }
func (p *Player) StartGliding()              { p.data.gliding = true }
func (p *Player) Gliding() bool              { return p.data.gliding }
func (p *Player) StopGliding()               { p.data.gliding = false }
func (*Player) FlightSpeed() float64         { return 0.05 }
func (*Player) VerticalFlightSpeed() float64 { return 1 }

func (p *Player) Effect(effectType effect.Type) (effect.Effect, bool) {
	for _, current := range p.Effects() {
		if current.Type() == effectType {
			return current, true
		}
	}
	return effect.Effect{}, false
}
func (p *Player) SwingArm() {
	for _, viewer := range p.Viewers() {
		viewer.ViewEntityAction(p, entity.SwingArmAction{})
	}
}
func (*Player) PunchAir()                    {}
func (*Player) Jump()                        {}
func (*Player) Exhaust(float64)              {}
func (*Player) Sleep(cube.Pos)               {}
func (*Player) Wake()                        {}
func (*Player) Respawn() *world.EntityHandle { return nil }

func (p *Player) AttackEntity(target world.Entity) bool {
	livingTarget, ok := target.(entity.Living)
	if !ok || livingTarget.Dead() {
		return false
	}
	_, slowFalling := p.Effect(effect.SlowFalling)
	_, blind := p.Effect(effect.Blindness)
	force, height := 0.45, 0.3608
	critical := !p.Sprinting() && !p.Flying() && p.FallDistance() > 0 && !slowFalling && !blind
	held, _ := p.HeldItems()
	if knockback, ok := held.Enchantment(enchantment.Knockback); ok {
		increase := enchantment.Knockback.Force(knockback.Level())
		force += increase
		height += increase
	}
	if p.data.handler.HandleAttackEntity(p, target, &force, &height, &critical) {
		return false
	}
	damage := held.AttackDamage()
	if strength, ok := p.Effect(effect.Strength); ok {
		damage += damage * effect.Strength.Multiplier(strength.Level())
	}
	if weakness, ok := p.Effect(effect.Weakness); ok {
		damage -= damage * effect.Weakness.Multiplier(weakness.Level())
	}
	if sharpness, ok := held.Enchantment(enchantment.Sharpness); ok {
		damage += enchantment.Sharpness.Addend(sharpness.Level())
	}
	if critical {
		damage *= 1.5
	}
	_, vulnerable := livingTarget.Hurt(damage, entity.AttackDamageSource{Attacker: p})
	p.SwingArm()
	if vulnerable {
		livingTarget.KnockBack(p.Position(), force, height)
	}
	return true
}

func (p *Player) UseItem() {
	held, off := p.HeldItems()
	usable, ok := held.Item().(item.Usable)
	if !ok {
		return
	}
	ctx := &item.UseContext{}
	if !usable.Use(p.Tx(), p, ctx) {
		return
	}
	p.SwingArm()
	if ctx.CountSub > 0 {
		held = held.Grow(-ctx.CountSub)
	}
	p.SetHeldItems(held, off)
}
func (p *Player) UsingItem() bool                              { return p.data.usingItem }
func (p *Player) ReleaseItem()                                 { p.data.usingItem = false }
func (*Player) UseItemOnBlock(cube.Pos, cube.Face, mgl64.Vec3) {}
func (*Player) UseItemOnEntity(world.Entity) bool              { return false }
func (*Player) BreakBlock(cube.Pos)                            {}
func (*Player) PickBlock(cube.Pos)                             {}
func (p *Player) Drop(stack item.Stack) int                    { return stack.Count() }

func (p *Player) SetCooldown(value world.Item, duration time.Duration) {
	name, _ := value.EncodeItem()
	p.data.cooldowns[name] = time.Now().Add(duration)
}
func (p *Player) SetNameTag(name string)                { p.Living.SetNameTag(name) }
func (*Player) MoveItemsToInventory()                   {}
func (*Player) StartBreaking(cube.Pos, cube.Face)       {}
func (*Player) ContinueBreaking(cube.Face)              {}
func (*Player) FinishBreaking()                         {}
func (*Player) AbortBreaking()                          {}
func (*Player) OpenSign(cube.Pos, bool)                 {}
func (*Player) EditSign(cube.Pos, string, string) error { return nil }
func (*Player) TurnLecternPage(cube.Pos, int) error     { return nil }

func (*Player) SendDialogue(dialogue.Dialogue, world.Entity) {}
func (*Player) CloseDialogue()                               {}
func (*Player) SendForm(form.Form)                           {}
func (*Player) CloseForm()                                   {}
func (*Player) SendCommandOutput(*cmd.Output)                {}
func (*Player) Message(...any)                               {}
func (*Player) Chat(...any)                                  {}
func (*Player) ExecuteCommand(string)                        {}
func (*Player) ShowHudElement(hud.Element)                   {}
func (*Player) HideHudElement(hud.Element)                   {}
func (*Player) HudElementHidden(hud.Element) bool            { return false }
func (*Player) AddDebugShape(debug.Shape)                    {}
func (*Player) RemoveDebugShape(debug.Shape)                 {}
func (*Player) VisibleDebugShapes() []debug.Shape            { return nil }
func (*Player) RemoveAllDebugShapes()                        {}
func (*Player) LockInput(input.Lock)                         {}
func (*Player) UnlockInput(input.Lock)                       {}
func (*Player) ClearInputLocks()                             {}
func (*Player) InputLocked(input.Lock) bool                  { return false }
func (*Player) UpdateDiagnostics(session.Diagnostics)        {}

func (p *Player) Viewers() []world.Viewer { return p.Tx().Viewers(p.Position()) }
