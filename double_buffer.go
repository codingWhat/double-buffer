package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"time"
)

func main() {

	file, err := os.OpenFile("./1.txt", os.O_APPEND|os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		panic(err.Error())
	}

	dBuffer := DoubleBuffer{
		file: file,
	}

	for i := 0; i < 2; i++ {
		var buf []byte
		dBuffer.container[i] = bytes.NewBuffer(buf)
	}

	ticker := time.NewTimer(200 * time.Second)

	for i := 0; i < 10; i++ {
		go func(idx int) {

			for {

				dBuffer.Lock()
				fmt.Println("++++++++协程", idx, "抢到锁")
				dBuffer.container[dBuffer.curIdx].Write([]byte(fmt.Sprintf("test:%d---%d，---curIdx:%d \n，", idx,
					rand.Int(), dBuffer.curIdx)))
				//20 M
				flag := false
				if dBuffer.container[dBuffer.curIdx].Len() > 2*1024 {
					//切换内存区
					dBuffer.curIdx = (dBuffer.curIdx + 1) % 2
					flag = true

				}
				dBuffer.Unlock()

				//判断写入磁盘
				if flag {
					writeIdx := dBuffer.GetNextWriteIdx()
					fmt.Println("========协程", idx, "准备写入文件")
					f := fmt.Sprintf("========协程 %d 准备写入文件, len:%d \n", idx, dBuffer.container[dBuffer.curIdx].Len())
					final := []byte(f)
					final = append(final, dBuffer.container[writeIdx].Bytes()...)
					_, err := dBuffer.file.Write(final)
					if err != nil {
						panic(err.Error())
					}
					dBuffer.container[writeIdx].Reset()
				}
			}

		}(i)
	}

	for {

		select {
		case <-ticker.C:
			fmt.Println("时钟结束")
			return
		}

	}

}

//type DoubleBuffer struct {
//	sync.RWMutex
//	file      *os.File
//	curIdx    int
//	container [2]*bytes.Buffer
//}
//
//func (d *DoubleBuffer) GetCurIdx() int {
//	d.RLock()
//	defer d.RUnlock()
//	return d.curIdx
//}
//
//func (d *DoubleBuffer) GetNextWriteIdx() int {
//	d.RLock()
//	defer d.RUnlock()
//	return (d.curIdx + 1) % 2
//}
