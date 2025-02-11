package event_processor

import (
    "edemo/user/event"
    "fmt"
    "log"
    "sync"
)

const (
    MAX_INCOMING_CHAN_LEN = 1024
    MAX_PARSER_QUEUE_LEN  = 1024
)

type EventProcessor struct {
    sync.Mutex
    // 收包，来自调用者发来的新事件
    incoming chan event.IEventStruct

    // key为 PID+UID+COMMON等确定唯一的信息
    workerQueue map[string]IWorker

    logger *log.Logger

    // output model
    isHex bool
}

func (this *EventProcessor) GetLogger() *log.Logger {
    return this.logger
}

func (this *EventProcessor) init() {
    this.incoming = make(chan event.IEventStruct, MAX_INCOMING_CHAN_LEN)
    this.workerQueue = make(map[string]IWorker, MAX_PARSER_QUEUE_LEN)
}

// Write event 处理器读取事件
func (this *EventProcessor) Serve() {
    for {
        select {
        case e := <-this.incoming:
            // 从channel中读取解析好的事件
            this.dispatch(e)
        }
    }
}

func (this *EventProcessor) dispatch(e event.IEventStruct) {

    var uuid string = e.GetUUID()
    found, eWorker := this.getWorkerByUUID(uuid)
    if !found {
        // ADD a new eventWorker into queue
        eWorker = NewEventWorker(e.GetUUID(), this)
        this.addWorkerByUUID(eWorker)
    }
    // 调用 worker 正式处理解析好的事件数据
    err := eWorker.Write(e)
    if err != nil {
        //...
        this.GetLogger().Fatalf("write event failed , error:%v", err)
    }
}

//func (this *EventProcessor) Incoming() chan user.IEventStruct {
//    return this.incoming
//}

func (this *EventProcessor) getWorkerByUUID(uuid string) (bool, IWorker) {
    this.Lock()
    defer this.Unlock()
    var eWorker IWorker
    var found bool
    eWorker, found = this.workerQueue[uuid]
    if !found {
        return false, eWorker
    }
    return true, eWorker
}

func (this *EventProcessor) addWorkerByUUID(worker IWorker) {
    this.Lock()
    defer this.Unlock()
    this.workerQueue[worker.GetUUID()] = worker
}

// 每个worker调用该方法，从处理器中删除自己
func (this *EventProcessor) delWorkerByUUID(worker IWorker) {
    this.Lock()
    defer this.Unlock()
    delete(this.workerQueue, worker.GetUUID())
}

// Write event
// 外部调用者调用该方法
func (this *EventProcessor) Write(e event.IEventStruct) {
    // 事件解析后走到这里
    select {
    case this.incoming <- e:
        return
    }
}

func (this *EventProcessor) Close() error {
    if len(this.workerQueue) > 0 {
        return fmt.Errorf("EventProcessor.Close(): workerQueue is not empty:%d", len(this.workerQueue))
    }
    return nil
}

func NewEventProcessor(logger *log.Logger, isHex bool) *EventProcessor {
    var ep *EventProcessor
    ep = &EventProcessor{}
    ep.logger = logger
    ep.isHex = isHex
    ep.init()
    return ep
}
