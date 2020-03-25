package main

import (
	"fmt"
	"bufio"
	"os"
)

func main() {
	myActor := ActorOf(new(MyActor))
	myActor.Tell(Hello{Name: "Roger"})
	myActor.Tell(Hello{Name: "Go"})
	myActor.Tell(100)
	bufio.NewReader(os.Stdin).ReadString('\n')
}

// ActorRef 框架的使用接口
type ActorRef interface {
	Tell(message interface{})
	SendSystemMessage(message interface{})
}

// ChannelActorRef 用通道Channel来实现消息的投递
type ChannelActorRef struct {
	actorCell *ActorCell
}

func (ref *ChannelActorRef) Tell(message interface{}) {
	ref.actorCell.userMailbox <- message
}

func (ref *ChannelActorRef) SendSystemMessage(message interface{}) {
	ref.actorCell.userMailbox <- message
}

// Actor 真正接收消息的处理部分
type Actor interface {
	Receive(message interface{})
}

// ActorOf 有两个功能
// 1. 调度的初始化，以及ActorRef的构建
// 2. 调度部分，消息的接收与分发传递处理
func ActorOf(actor Actor) ActorRef {
	userMailbox := make(chan interface{}, 100)
	systemMailbox := make(chan interface{}, 100)
	cell := &ActorCell{
		userMailbox:	userMailbox,
		systemMailbox:	systemMailbox,
		actor:		actor,
	}
	ref := ChannelActorRef{
		actorCell: cell,
	}

	go func() {
		for {
			select {
			case sysMsg := <-systemMailbox:
				// prioritize sys messages
				cell.invokeSystemMessage(sysMsg)
			default:
				// if no system message is present, try read user message
				select {
				case userMsg := <-userMailbox:
					cell.invokeUserMessage(userMsg)
				default:
				}
			}
		}
	} ()

	return &ref
}

// ActorCell 一个中间部件，连接调度部分与Actor部分
type ActorCell struct {
	userMailbox	chan interface{}
	systemMailbox	chan interface{}
	actor		Actor
}

func (cell *ActorCell) invokeSystemMessage(message interface{}) {
	fmt.Printf("Received system message %v\n", message)
}

func (cell *ActorCell) invokeUserMessage(message interface{}) {
	cell.actor.Receive(message)
}

// ----- Demo -----

type MyActor struct{ messageCount int }
type Hello struct{ Name string }

func (state *MyActor) Receive(message interface{}) {
	switch msg := message.(type) {
	default:
		fmt.Printf("unexpected type %T\n", msg) // %T prints whatever type t has //whl?
	case Hello:
		fmt.Printf("Hello %v\n", msg.Name) // t has type bool //whl?
		state.messageCount++
	case int:
		fmt.Printf("Come on, number %v", msg)
	}
}
