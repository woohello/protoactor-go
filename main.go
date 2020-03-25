package main

import (
	"fmt"
	"bufio"
	"os"
	"sync/atomic"
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
	ref.actorCell.mailbox.userMailbox <- message
	ref.actorCell.schedule()
}

func (ref *ChannelActorRef) SendSystemMessage(message interface{}) {
	ref.actorCell.mailbox.userMailbox <- message
	ref.actorCell.schedule()
}

// Actor 真正接收消息的处理部分
type Actor interface {
	Receive(message interface{})
}

// ActorOf 调度的初始化，以及ActorRef的构建
func ActorOf(actor Actor) ActorRef {
	userMailbox := make(chan interface{}, 100)
	systemMailbox := make(chan interface{}, 100)
	mailbox := Mailbox{
		userMailbox:	userMailbox,
		systemMailbox:	systemMailbox,
	}
	cell := &ActorCell{
		mailbox:	&mailbox,
		actor:		actor,
		hasMoreMessages:int32(0),
		schedulerStatus:int32(0),
	}
	ref := ChannelActorRef{
		actorCell: cell,
	}


	return &ref
}

const MailboxIdle int32 = 0
const MailboxBusy int32 = 1
const MailboxHasMoreMessages int32 = 1
const MailboxHasNoMessages int32 = 0

func (cell *ActorCell) schedule() {
	// 这里的调度，会先通过原子操作CAS，来做同步的判断，判断schedulerStatus是否为Idle，是则切换成Busy，否则不切换。
	swapped := atomic.CompareAndSwapInt32(&cell.schedulerStatus, MailboxIdle, MailboxBusy)
	atomic.StoreInt32(&cell.hasMoreMessages, MailboxHasMoreMessages) // we have more messages to process
	if swapped {
		go cell.processMessages()
	}
}

func (cell *ActorCell) processMessages() {
	atomic.StoreInt32(&cell.hasMoreMessages, MailboxHasNoMessages)
	for i:=0; i<30; i++ {
		select {
		case sysMsg := <-cell.mailbox.systemMailbox:
			// prioritize system messages
			cell.invokeSystemMessage(sysMsg)
		default:
			// if no system messages is presend, try read user message
			select {
			case userMsg := <-cell.mailbox.userMailbox:
				cell.invokeUserMessage(userMsg)
			default:
			}
		}
	}
	atomic.StoreInt32(&cell.schedulerStatus, MailboxIdle)
	// -----begin: double check, 防止上面两行代码之间有人调用schedule而被忽略掉。这个是关键点。
	// was there any messages scheduler since we began processing?
	hasMore := atomic.LoadInt32(&cell.hasMoreMessages)
	status := atomic.LoadInt32(&cell.schedulerStatus)
	// have there been any scheduling of mailbox? (e.g. race condition from the two above lines
	if hasMore == MailboxHasMoreMessages && status == MailboxIdle {
		swapped := atomic.CompareAndSwapInt32(&cell.schedulerStatus, MailboxIdle, MailboxBusy)
		if swapped {
			go cell.processMessages()
		}
	}
	// -----end
}

// ActorCell 一个中间部件，连接调度部分与Actor部分
type ActorCell struct {
	mailbox		*Mailbox
	actor		Actor
	schedulerStatus int32
	hasMoreMessages int32
}

type Mailbox struct {
	userMailbox	chan interface{}
	systemMailbox	chan interface{}
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
	}
}
