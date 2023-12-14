package living

type Handler interface {
	// HandleTick handles the entity's tick.
	HandleTick(ent *Living)
}

type NopHandler struct{}

var _ Handler = NopHandler{}

func (NopHandler) HandleTick(*Living) {}
