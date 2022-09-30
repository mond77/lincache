package singleflight

import "sync"

type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

type Group struct {
	mu sync.Mutex
	mm map[string]*call

}

//
func (g *Group) Do(key string,fn func()(interface{},error))(interface{},error){
	g.mu.Lock()
	//延迟初始化，提高内存使用效率
	if g.mm == nil{
		g.mm = make(map[string]*call)
	}

	if c,ok := g.mm[key];ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val,c.err
	}
	c := new(call)
	c.wg.Add(1)	//发起请求前加锁
	g.mm[key] = c
	g.mu.Unlock()

	c.val,c.err = fn()
	c.wg.Done()//请求结束

	//只有在请求期间发起的同一个key查询
	g.mu.Lock()
	delete(g.mm,key)
	g.mu.Unlock()

	return c.val,c.err
}