package living

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"math/rand/v2"
)

type Drop struct {
	it       world.Item
	min, max int
}

func NewDrop(it world.Item, min, max int) Drop {
	return Drop{
		it:  it,
		min: min,
		max: max,
	}
}

func (d Drop) Stack() item.Stack {
	c := rand.IntN(d.max-d.min) + d.min
	if c == 0 {
		return item.Stack{}
	}
	return item.NewStack(d.it, c)
}
