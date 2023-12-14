package main

import (
	"github.com/bedrock-gophers/living/living"
	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/chat"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sirupsen/logrus"
	"math/rand"
)

func main() {
	log := logrus.New()
	log.Formatter = &logrus.TextFormatter{ForceColors: true}
	log.Level = logrus.DebugLevel

	chat.Global.Subscribe(chat.StdoutSubscriber{})

	conf, err := server.DefaultConfig().Config(log)
	if err != nil {
		log.Fatalln(err)
	}

	srv := conf.New()
	srv.CloseOnProgramEnd()

	srv.Listen()
	for srv.Accept(accept) {

	}
}

func accept(p *player.Player) {
	enderman := living.NewLivingEntity(entityTypeEnderman{}, 40, 0.3, []item.Stack{item.NewStack(item.EnderPearl{}, rand.Intn(2)+1)}, &entity.MovementComputer{
		Gravity:           0.08,
		Drag:              0.02,
		DragBeforeGravity: true,
	}, p.Position(), p.World())

	p.World().AddEntity(enderman)
}

type entityTypeEnderman struct{}

func (entityTypeEnderman) EncodeEntity() string {
	return "minecraft:enderman"
}
func (entityTypeEnderman) BBox(world.Entity) cube.BBox {
	return cube.Box(-0.3, 0, -0.3, 0.3, 2.9, 0.3)
}
