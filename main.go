package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

func Walk(file *os.File, path string, m *sync.Mutex, wg *sync.WaitGroup) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	data, err := dir.Readdir(-1)
	if err != nil {
		return err
	}

	for _, d := range data {
		if !d.IsDir() {
			m.Lock()
			_, err = file.WriteString(dir.Name() + "; " + d.Name() + "; " + strconv.FormatInt(d.Size(), 10) + "\n")
			m.Unlock()
			if err != nil {
				return err
			}
			continue
		}
		wg.Add(1)
		go Walk(file, path+"/"+d.Name(), m, wg)
		if err != nil {
			return err
		}
	}
	wg.Done()
	return nil
}

func Walksync(file *os.File, path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	data, err := dir.Readdir(-1)
	if err != nil {
		return err
	}

	for _, d := range data {
		if !d.IsDir() {
			_, err = file.WriteString(dir.Name() + "; " + d.Name() + "; " + strconv.FormatInt(d.Size(), 10) + "\n")
			if err != nil {
				return err
			}
			continue
		}
		Walksync(file, path+"/"+d.Name())
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	path := "H:/" // Записываем путь к интерисующей дериктории
	file, err := os.OpenFile("test.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	//---------------------------
	/* t := time.Now()
	fmt.Println(t)
	err = Walksync(file, path)
	if err != nil {
		panic(err)
	}
	t2 := time.Now()
	fmt.Println(t2.Sub(t)) */
	//---------------------------
	runtime.GOMAXPROCS(1000)
	var m sync.Mutex
	var wg sync.WaitGroup
	t3 := time.Now()
	fmt.Println(t3)
	err = Walk(file, path, &m, &wg)
	wg.Wait()
	if err != nil {
		panic(err)
	}
	t4 := time.Now()
	fmt.Println(t4.Sub(t3))
}
