# Living

The Living library provides tools for creating and managing living entities within the Minecraft world. Whether you're spawning custom mobs or adding unique behaviors to existing ones, Living offers an easy-to-use framework for implementing complex entity logic.

# Importing Living into your project:

You may import Living by running the following command:
```bash
go get github.com/bedrock-gophers/living
```

## Adding a Living Entity to Your World

To add a living entity to your world, you can use the following example code:

```go
func main() {
    // Example code to add an Enderman entity to the world
    enderman := living.NewLivingEntity(entityTypeEnderman{}, 40, 0.3, []item.Stack{item.NewStack(item.EnderPearl{}, rand.Intn(2)+1)}, &entity.MovementComputer{
    	Gravity:           0.08,
    	Drag:              0.02,
    	DragBeforeGravity: true,
    }, p.Position(), p.World())
    
    enderman.SetNameTag("Enderman")
    enderman.Handle(handler{e: enderman})
    
    p.World().AddEntity(enderman)
}
```

## Creating and Handling a living entity
To create and handle a living entity, you can use the following example code:

```go
// Define a custom entity type for Enderman.
type entityTypeEnderman struct{}

// EncodeEntity ...
func (entityTypeEnderman) EncodeEntity() string {
	return "minecraft:enderman"
}

// BBox ...
func (entityTypeEnderman) BBox(world.Entity) cube.BBox {
	return cube.Box(-0.3, 0, -0.3, 0.3, 2.9, 0.3)
}

// handler represents a basic event handler for our endermen.
type handler struct {
	living.NopHandler
	e *living.Living
}

// HandleHurt ...
func (handler) HandleHurt(ctx *event.Context, damage float64, src world.DamageSource) {
	fmt.Println("enderman hurt")
}
```

This code defines an example of creating an Enderman entity type and implementing a custom event handler for handling hurt events. You can extend this pattern to implement various other behaviors and interactions for your living entities.
