package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"
)

func main() {

	dbs := NewDoubleBufferSet(10) // 10个内存分片，降低锁粒度

	for i := 0; i < 20; i++ {

		go func(idx int) {

			for  {
				//随机一个内存分片 , 负载均衡
				dBuffer := dbs.GetRandomDB()

				//内存分片加锁
				dBuffer.Lock()
				fmt.Println("协程", idx, "抢到锁")
				dBuffer.container[dBuffer.curIdx].Write([]byte(fmt.Sprintf("test:%d---%d，---curIdx:%d \n", idx,
					rand.Int(), dBuffer.curIdx)))

				flag := false
				//如果内存分片1分区写满了
				if dBuffer.container[dBuffer.curIdx].Len() > 2 {
					//切换可写空间
					dBuffer.curIdx = (dBuffer.curIdx + 1 ) % 2
					flag = true

				}

				dBuffer.Unlock()  //这里很重要,要及时释放锁,将耗时长的刷磁盘,挪出加锁逻辑
				//如果分区1写满了，那么当前协程执行刷盘操作，而其他协程能继续写分区2
				if flag {
					writeIdx := dBuffer.GetNextWriteIdx()
					f := fmt.Sprintf("========协程 %d 准备写入文件, len:%d \n", idx, dBuffer.container[dBuffer.curIdx].Len())
					final := []byte(f)
					final = append(final, dBuffer.container[writeIdx].Bytes()...)
					_, err := dBuffer.file.Write(final)
					if err != nil {
						panic(err.Error())
					}
					dBuffer.container[writeIdx].Reset()
				}

				s := rand.Int()
				time.Sleep(time.Duration(s % 10) * time.Second)
			}


		}(i)

	}

	timer := time.NewTimer(100 * time.Second)

	for  {
		select {
		case <-timer.C:
			fmt.Println("timer stop")
			return

		}
	}

}

type DoubleBufferSet struct {
	data []*DoubleBuffer
	mark  int
}

func NewDoubleBufferSet(partitionNum int) *DoubleBufferSet {

	dbs := &DoubleBufferSet{
		data: make([]*DoubleBuffer, partitionNum),
		mark: partitionNum,
	}

	for i := 0; i < partitionNum; i++ {

		fileName := fmt.Sprintf("./log_%d.txt", i+1)
		file, err := os.OpenFile(fileName, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			panic(err.Error())
		}

		dbs.data[i] = &DoubleBuffer{
			file: file,
		}
		for j := 0; j < 2; j++ {
			var buf []byte
			dbs.data[i].container[j] = bytes.NewBuffer(buf)
		}
	}


	return dbs
}

func (db *DoubleBufferSet) GetRandomDB() *DoubleBuffer{
	return db.data[rand.Int() % db.mark]
}


type DoubleBuffer struct {
	sync.RWMutex
	file      *os.File
	curIdx    int
	container [2]*bytes.Buffer
}

func (d *DoubleBuffer) GetCurIdx() int {
	d.RLock()
	defer d.RUnlock()
	return d.curIdx
}

func (d *DoubleBuffer) GetNextWriteIdx() int {
	d.RLock()
	defer d.RUnlock()
	return (d.curIdx + 1) % 2
}
