package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func WalkWithOutRecursion(file *os.File, path string) error {
	stack := []string{path}              //стэк папок
	limit := time.Tick(time.Millisecond) // промежуто для остановки перед записью
	// Пока стэк не пустой, достаем из него папку и проходим по ней
	for len(stack) > 0 {
		dirPath := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		/* if dir == "." {
		       continue
		   }
		   if dir == "/" {
		       continue
		   } */
		dir, err := os.Open(dirPath)
		if err != nil {
			return err
		}
		defer dir.Close()
		filesInfo, err := dir.Readdir(0)
		if err != nil {
			return err
		}
		for _, fileInfo := range filesInfo {
			if fileInfo.Name() == "System Volume Information" {
				continue
			}
			if fileInfo.IsDir() {
				nextPath := filepath.Join(dirPath, fileInfo.Name())
				stack = append(stack, nextPath)
				continue
			}
			<-limit //лимит перед записью в 10ms
			_, err := file.WriteString(dirPath + "; " + fileInfo.Name() + "; " + strconv.FormatInt(fileInfo.Size(), 10) + "\n")
			if err != nil {
				return err
			}
		}
	}
	return nil
}
func main() {
	path := `C:\Users\lebed` // Записываем путь к интересующей дериктории
	//path := `C:\Program Files (x86)\Steam` // Записываем путь к интересующей дериктории
	//path := `/media/tiltamen/66666FD8666FA80F/Users/lebed`
	//path := `/media/tiltamen/66666FD8666FA80F/Windows`
	file, err := os.OpenFile("test.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600) // открываем файл для записи
	if err != nil {
		panic(err)
	}
	defer file.Close()
	//---------------------------Обход без рекурсии
	t1 := time.Now()
	fmt.Println(t1)
	err = WalkWithOutRecursion(file, path)
	if err != nil {
		panic(err)
	}
	t2 := time.Now()
	fmt.Println(t2.Sub(t1))
	//---------------------------Обход с остановками
	/* t3 := time.Now()
	fmt.Println(t3)
	err = WalkPathStop(file, path, 5000)
	if err != nil {
		panic(err)
	}
	t4 := time.Now()
	fmt.Println(t4.Sub(t3)) */
	//---------------------------Последовательный обход
	/* runtime.GOMAXPROCS(5)
	t5 := time.Now()
	fmt.Println(t5)
	err = WalkPath(file, path)
	if err != nil {
		panic(err)
	}
	t6 := time.Now()
	fmt.Println(t6.Sub(t5)) */
	//---------------------------Попытка в асинхронный обход
	/* t7 := time.Now()
	fmt.Println(t7)
	err = WalkAsync(file, path, 100)
	if err != nil {
		panic(err)
	}
	t8 := time.Now()
	fmt.Println(t8.Sub(t7)) */
	//! Всего файлов в моем HDD набралось 215506
	//! время работы последовательного метода:
	//! windows - 2.23 - 2.25 мин.
	//! linux - 1.54 - 2.54 мин.
	//! время работы асинхронного метода:
	//! windows - 2.37 - 2.15 мин.
	//! linux - 1.14 - 3.56 мин.
	//! Каждый раз перед запуском нового теста HDD отключался от компьютера
	//! Тесты запускались отдельно друг от друга
	return
}
