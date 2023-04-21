package main

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// Функция с Асинхронным обходом файлов дериктории
// file - файл в который будет вестись запись, path - путь к дериктории которую будем обходить
// workers - максимальное количество горутин на обход дериктории
func WalkAsync(file *os.File, path string, workers int) error {
	// создаем канал для записи в файл
	resChan := make(chan string)
	var wg sync.WaitGroup
	// запускаем множество горутин записи в файл
	//* скорее всего операция бессмысленная поскольку в момент времени 2 горутины не могут писать в файл
	//* с другой стороны это необходимо чтобы канал на запись не заблокировался
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for res := range resChan {
				_, err := file.WriteString(res)
				if err != nil {
					log.Fatal(err)
				}
			}
		}()
	}

	wg.Add(1)
	var wg2 sync.WaitGroup
	workChan := make(chan int, workers)
	// запускаем множество горутин обхода дериктории
	// чтобы они не завершились раньше времени, пришлось добавить второй объект waitGroup
	// и дополнительный канал для работающих горутин, чтобы не выйти за пределы работающих потоков
	go func(wg2 *sync.WaitGroup) {
		defer wg.Done()
		wg2.Add(1)
		workChan <- 1
		walkDir(path, resChan, wg2, workChan)
		wg2.Wait()
		close(resChan)
	}(&wg2)
	wg.Wait()
	close(workChan)
	return nil
}

func walkDir(dir string, fileChan chan<- string, wg *sync.WaitGroup, workChan chan int) {
	defer wg.Done()

	entries, err := ioutil.ReadDir(dir) // Прочитываем все список содержимого каталога
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, entry := range entries {
		//! я делал проверку работы алгоритма на съемном HDD
		//! обработать системный файл не получалось из-за недостатка прав
		if entry.Name() == "System Volume Information" {
			continue
		}
		// если рассматриваемый объект является дерикторией
		if entry.IsDir() {
			path := filepath.Join(dir, entry.Name())
			wg.Add(1)
			// проверяем переполнен ли пул рабочих потоков
			// если нет, запускаем асинхронный обход дериктории
			// если да, запускаем в этой же горутине обход дериктории
			select {
			case workChan <- 1:
				go walkDir(path, fileChan, wg, workChan) // Запускаем обход поддиректории в отдельной горутине
			default:
				walkDir(path, fileChan, wg, workChan) // Запускаем обход поддиректории в отдельной горутине
			}
		} else {
			// если рассматриваемый объект является файлом - записываем его данные в файл
			fileChan <- dir + "; " + entry.Name() + "; " + strconv.FormatInt(entry.Size(), 10) + "\n" // Отправляем путь файла в канал
		}
	}
	// по окончанию обхода читаем из пула работников значение для освобождения места под параллельный процесс
	select {
	case <-workChan:
		return
	default:
		return
	}

}

// Функция с Последовательным обходом файлов дериктории
// file - файл в который будет вестись запись, path - путь к дериктории которую будем обходить
func WalkPath(file *os.File, path string) error {
	dataChan := make(chan string)
	var wg sync.WaitGroup
	limit := time.Tick(time.Millisecond * 10) // промежуто для остановки перед записью и перед переходом к следующему файлу

	// аналогично Асинхронному методу создаем горктины на запись
	for i := 0; i < 1; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-limit
			for fileData := range dataChan {
				_, err := file.WriteString(fileData)
				if err != nil {
					log.Fatal(err)
				}
			}
		}()
	}
	// с помощью библитеки filepath обходим дерикторию в ложенные подпапки
	err := filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		//если рассматриваемый обЪект является дерикторией
		// просто идем дальше, поскольку метод Walk() сам обходит дериктории, ему не нужны указания на это
		if info.IsDir() {
			return nil
		}
		// если рассматриваемый объект является файлом - записываем результат
		dataChan <- path + "; " + info.Name() + "; " + strconv.FormatInt(info.Size(), 10) + "\n"
		return nil
	})
	if err != nil {
		return err
	}
	close(dataChan)
	wg.Wait()
	return nil
}

// Функция с Последовательным обходом файлов дериктории
// file - файл в который будет вестись запись,
// path - путь к дериктории которую будем обходить
// maxProcLimit - максимальное колличество запускаемых горутин
func WalkPathStop(file *os.File, path string, maxProcLimit int) error {
	var wg sync.WaitGroup                        // необходимо для ожидания выполнения всех горутин
	maxProc := make(chan struct{}, maxProcLimit) // максимальное колличество запускаемых горутин
	limit := time.Tick(time.Nanosecond * 5)      // промежуто для остановки перед записью и перед переходом к следующему файлу
	var dataChan = make(chan string, 300)        // канал для передачи данных о файле

	// функция для записи в файл
	writer := func() {
		defer wg.Done()
		for data := range dataChan {
			<-limit
			_, err := file.WriteString(data)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	// запускаем много, чтобы выполнение долго не простаивало
	for i := 0.0; i < 290; i++ {
		<-limit
		wg.Add(1)
		go writer()
	}

	// ф-ция для обхода по файлам
	var wg2 sync.WaitGroup // необходимо для ожидания выполнения всех горутин
	var travelDir func(string)
	travelDir = func(dir string) {
		defer wg2.Done()

		dirFiles, err := os.Open(dir) // открываем дерикторию
		if err != nil {
			log.Fatal(err)
		}
		defer dirFiles.Close()
		fileInfos, err := dirFiles.Readdir(0) // считываем все файлы из дериктории
		if err != nil {
			log.Fatal(err)
		}
		for _, fileInfo := range fileInfos {
			//* приостановка выполнения, полагаю можно без нее, но не уверен
			if fileInfo.Name() == "System Volume Information" { //игонрирование системного файла
				continue
			}
			// если рассматриваемый объект является папкой
			if fileInfo.IsDir() {
				nextPath := filepath.Join(dir, fileInfo.Name())
				wg2.Add(1)
				// проверяем переполнен ли пул рабочих
				if len(maxProc) == cap(maxProc) {
					travelDir(nextPath)
					continue
				}
				<-limit
				// если не переполнен то делаем новую горутину
				go travelDir(nextPath)
				continue
			}
			dataChan <- dir + "; " + fileInfo.Name() + "; " + strconv.FormatInt(fileInfo.Size(), 10) + "\n"
		}
		// освобождение места под новые горутины
		select {
		case <-maxProc:
			return
		default:
			return
		}
	}
	wg2.Add(1)
	go travelDir(path)

	wg2.Wait()
	close(dataChan)
	wg.Wait()
	return nil
}

func main() {
	path := "H:/"                                                                      // Записываем путь к интерисующей дериктории
	file, err := os.OpenFile("H:/test.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600) // открываем файл для записи
	if err != nil {
		panic(err)
	}
	defer file.Close()
	//---------------------------Обход с остановками
	t3 := time.Now()
	fmt.Println(t3)
	err = WalkPathStop(file, path, 5000)
	if err != nil {
		panic(err)
	}
	t4 := time.Now()
	fmt.Println(t4.Sub(t3))
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
