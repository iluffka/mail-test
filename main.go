package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
)

func main() {
	countGR := make(chan struct{}, 5) //k горутин
	sourceChan := make(chan string)
	countChan := make(chan int)

	var typeIn string
	flag.StringVar(&typeIn, "type", "url", "var from command line.")
	flag.Parse()

	f := setFunc(typeIn) //определяем тип операции

	go getInputData(sourceChan, typeIn)
	go readData(sourceChan, countChan, countGR, f)
	countResult(countChan)
}

func getInputData(sourceChan chan<- string, typeIn string) {
	defer close(sourceChan)
	var urlReg = regexp.MustCompile(`http(s)?://[a-z0-9-]+(.[a-z0-9-]+)*(:[0-9]+)?(/.*)?`)
	//читаем данные с консоли
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		record := scanner.Text()
		lines := strings.Split(record, `\n`)
		for _, line := range lines {
			if prepareData(line, typeIn, urlReg) {
				sourceChan <- line //в канал sourceChan записываем либо url либо имя файла
			}
		}
	}
}

func setFunc(typeIn string) func(s string) (int, error) {
	//var f = func(s string) (i int, e error) {return}
	if typeIn != "url" && typeIn != "file" {
		log.Fatalln("type is not correct")
	}
	f := makeRequest
	if typeIn == "file" {
		f = getContentFromFile
	}

	return f
}

func readData(sourceChan <-chan string, count chan<- int, countGR chan struct{}, f func(s string) (int, error)) {
	var wg sync.WaitGroup

	for source := range sourceChan {
		wg.Add(1)
		countGR <- struct{}{}

		go func(source string) {
			defer wg.Done()
			cnt, err := f(source)
			if err != nil {
				log.Printf("fail to read file %s", err.Error())
				return
			}
			count <- cnt
			<-countGR
		}(source)
	}
	wg.Wait()
	close(count)
}

func getContentFromFile(fileName string) (int, error) {
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		return 0, err
	}
	count := strings.Count(string(content), "Go")
	fmt.Printf("Count for %s: %d\n", fileName, count)

	return count, nil
}

func makeRequest(url string) (int, error) {
	resp, err := http.Get(url)

	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return 0, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	content := string(body)

	count := strings.Count(content, "Go")
	fmt.Printf("Count for %s: %d\n", url, count)

	return count, nil
}

func countResult(countChan <-chan int) {
	count := 0
	for cnt := range countChan {
		count += cnt
	}
	fmt.Println("Total:", count)
}

func prepareData(str, typeIn string, urlReg *regexp.Regexp) bool {
	if typeIn == "file" {
		if _, err := os.Stat(str); os.IsNotExist(err) {
			log.Printf("file %s does not exist", str)
			return false
		}
	}
	if typeIn == "url" {
		str := urlReg.FindString(str)
		if _, err := url.ParseRequestURI(str); err != nil {
			log.Println(err)
			return false
		}
	}

	return true
}
