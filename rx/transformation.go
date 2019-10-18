package rx

import "sync"

//Map 对元素做一层映射（转换）
func (ob Observable) Map(f func(interface{}) interface{}) Observable {
	return func(sink *Observer) error {
		return ob(FuncObserver(func(event *Event) {
			sink.Next(f(event.Data))
		}, sink))
	}
}

//MapTo 映射成一个固定值
func (ob Observable) MapTo(data interface{}) Observable {
	return func(sink *Observer) error {
		return ob(FuncObserver(func(event *Event) {
			sink.Next(data)
		}, sink))
	}
}

//MergeMap 将元素映射成事件流并合并其中的元素
func (ob Observable) MergeMap(f func(interface{}) Observable, resultSelector func(interface{}, interface{}) interface{}) Observable {
	return func(sink *Observer) (err error) {
		wg := sync.WaitGroup{}
		wg.Add(1) //母事件流等待信号
		subMap := func(data interface{}) {
			f(data)(FuncObserver(sink.Push, sink))
			wg.Done()
		}
		if resultSelector != nil { //使用结果选择器
			subMap = func(data interface{}) {
				f(data)(FuncObserver(func(event *Event) {
					sink.Next(resultSelector(data, event.Data))
				}, sink))
				wg.Done()
			}
		}
		go func() {
			err = ob(FuncObserver(func(event *Event) {
				wg.Add(1)
				go subMap(event.Data)
			}, sink))
			wg.Done()
		}()
		wg.Wait() //等待所有子事件流完成后再完成
		return
	}
}

//MergeMapTo  将元素映射成指定的事件流并合并其中的元素
func (ob Observable) MergeMapTo(then Observable, resultSelector func(interface{}, interface{}) interface{}) Observable {
	return ob.MergeMap(func(i interface{}) Observable {
		return then
	}, resultSelector)
}

//SwitchMap 将元素映射成事件流然后发送其中的元素
func (ob Observable) SwitchMap(f func(interface{}) Observable, resultSelector func(interface{}, interface{}) interface{}) Observable {
	return func(sink *Observer) error {
		var currentSub *Observer
		wg := sync.WaitGroup{}
		subMap := func(data interface{}, currentSub *Observer) {
			f(data)(currentSub)
			wg.Done() //关闭Observer用来触发主事件流完成
		}
		wg.Add(1)
		go func() {
			ob(FuncObserver(func(event *Event) {
				if currentSub != nil { //关闭上一个子事件流
					currentSub.Dispose()
				}
				if resultSelector != nil {
					data := event.Data
					currentSub = FuncObserver(func(event *Event) {
						sink.Next(resultSelector(data, event.Data))
					}, nil)
				} else {
					currentSub = FuncObserver(sink.Push, nil)
				}
				wg.Add(1)
				go subMap(event.Data, currentSub)
			}, sink))
			wg.Done()
		}()
		wg.Wait()
		return sink.err
	}
}

//SwitchMapTo  将元素映射成指定的事件流然后发送其中的元素
func (ob Observable) SwitchMapTo(then Observable, resultSelector func(interface{}, interface{}) interface{}) Observable {
	return ob.SwitchMap(func(i interface{}) Observable {
		return then
	}, resultSelector)
}
