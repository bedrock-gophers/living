package living

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/world"
)

var Type = NopLivingType{}

type NopLivingType struct{}

func (n NopLivingType) Open(tx *world.Tx, handle *world.EntityHandle, data *world.EntityData) world.Entity {
	l := &Living{
		livingData:    data.Data.(*livingData),
		HealthManager: entity.NewHealthManager(20, 20),
		tx:            tx,
		handle:        handle,
		data:          data,
	}

	return l
}

func (NopLivingType) EncodeEntity() string {
	panic("implement me")
}

func (NopLivingType) BBox(e world.Entity) cube.BBox {
	return cube.Box(-0.3, 0, -0.3, 0.3, 2.9, 0.3)
}

func (NopLivingType) DecodeNBT(m map[string]any, data *world.EntityData) {

}

func (NopLivingType) EncodeNBT(data *world.EntityData) map[string]any {
	return map[string]any{}
}
