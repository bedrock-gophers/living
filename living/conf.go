package living

import (
	"slices"
	"time"

	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/world"
)

type Config struct {
	world.EntityType
	*entity.MovementComputer
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
