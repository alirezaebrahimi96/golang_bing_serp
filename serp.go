/*
Open a series of urls.

Check status code for each url and store urls I could not
open in a dedicated array.
Fetch urls concurrently using goroutines.
*/

package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/tealeg/xlsx/v3"
	"github.com/xuri/excelize/v2"
)

// -------------------------------------

// Custom user agent.
const (
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_6) "
)

// -------------------------------------

// fetchUrl opens a url with GET method and sets a custom user agent.
// If url cannot be opened, then log it to a dedicated channel.

func removeDuplicateStr(strSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

// A backoff schedule for when and how often to retry failed HTTP
// requests. The first element is the time to wait after the
// first failure, the second the time to wait after the second
// failure, etc. After reaching the last element, retries stop
// and the request is considered failed.
var backoffSchedule = []time.Duration{
	1 * time.Second,
	3 * time.Second,
}

func getURLDataWithRetries(url []string) {
	var err error

	for _, backoff := range backoffSchedule {

		// Create 2 channels, 1 to track urls we could not open
		// and 1 to inform url fetching is done:
		chFailedUrls := make(chan string)
		chIsFinished := make(chan bool)
		// Open all urls concurrently using the 'go' keyword:
		var links []string
		for _, url := range urlsList1 {
			time.Sleep(time.Millisecond * 50)
			go fetchUrl(url, chFailedUrls, chIsFinished, links)
		}

		// Receive messages from every concurrent goroutine. If
		// an url fails, we log it to failedUrls array:
		failedUrls := make([]string, 0)
		for i := 0; i < len(urlsList1); {
			select {
			case url := <-chFailedUrls:
				failedUrls = append(failedUrls, url)
			case <-chIsFinished:
				i++
			}
		}

		// Print all urls we could not open:
		fmt.Println("Could not fetch these urls: ", failedUrls)
		if err == nil {
			break
		}

		fmt.Fprintf(os.Stderr, "Request error: %+v\n", err)
		fmt.Fprintf(os.Stderr, "Retrying in %v\n", backoff)
		time.Sleep(backoff)
	}

}

func remove(slice []string, s int) []string {
	return append(slice[:s], slice[s+1:]...)
}

var keywords []string
var i int = 0

func fetchUrl(url string, chFailedUrls chan string, chIsFinished chan bool, name []string) {

	// Open url.
	// Need to use http.Client in order to set a custom user agent:
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", userAgent)
	resp, _ := client.Do(req)
	if resp != nil {
		doc, err := goquery.NewDocumentFromReader(resp.Body)
		// handle err
		if err != nil {
			log.Fatal(err)
		}
		tags1 := doc.Find(".b_ad b_adTop")
		tags1.Each(func(_ int, li *goquery.Selection) {
			links := li.Find("a")
			links.Each(func(_ int, a *goquery.Selection) {
				if val, ok := a.Attr("href"); ok {
					name = append(name, val)

				}
			})
		})
		tags2 := doc.Find(".b_algo")
		tags2.Each(func(_ int, li *goquery.Selection) {
			links := li.Find("a")
			links.Each(func(_ int, a *goquery.Selection) {
				if val, ok := a.Attr("href"); ok {
					name = append(name, val)
				}
			})
		})
		tags3 := doc.Find(".b_ad b_adBottom")
		tags3.Each(func(_ int, li *goquery.Selection) {
			links := li.Find("a")
			links.Each(func(_ int, a *goquery.Selection) {
				if val, ok := a.Attr("href"); ok {
					name = append(name, val)
					return
				}
			})
		})
		file_name := strconv.Itoa(i) + ".txt"

		file, err := os.OpenFile(file_name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

		if err != nil {
			log.Fatalf("failed creating file: %s", err)
		}

		datawriter := bufio.NewWriter(file)
		url = url[30 : len(url)-10]
		url = strings.ReplaceAll(url, "+", " ")
		keywords = append(words, url)

		_, _ = datawriter.WriteString(url + "\n")

		name := removeDuplicateStr(name)
		var links []string
		for i := 0; i < len(name); i++ {
			element := name[i]
			if strings.Contains(element, "https") {
				links = append(links, element)
			}
		}

		for _, data := range links {

			_, _ = datawriter.WriteString(data + "\n")

		}

		datawriter.Flush()
		file.Close()
		file2, err := os.OpenFile(file_name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

		if err != nil {
			log.Fatalf("failed creating file: %s", err)
		}
		file2.Close()
		fmt.Println(i / 10)
		i += 1

		//fetchLinks(links)
		// Inform the channel chIsFinished that url fetching is done (no
		// matter whether successful or not). Defer triggers only once
		// we leave fetchUrl():
		defer func() {
			chIsFinished <- true
		}()

		// If url could not be opened, we inform the channel chFailedUrls:
		if err != nil || resp.StatusCode != 200 {
			chFailedUrls <- url
			return
		}

	}

}

var words []string

func cellVisitor(c *xlsx.Cell) error {

	value, err := c.FormattedValue()
	if err != nil {
		fmt.Println(err.Error())
	} else {
		words = append(words, value)
	}
	return err
}

func rowVisitor(r *xlsx.Row) error {
	return r.ForEachCell(cellVisitor)
}

func rowStuff() {
	filename := "keywords.xlsx"
	wb, err := xlsx.OpenFile(filename)
	if err != nil {
		panic(err)
	}
	sh, ok := wb.Sheet["Sheet1"]
	if !ok {
		panic(errors.New("Sheet not found"))
	}
	fmt.Println("Max row is", sh.MaxRow)
	sh.ForEachRow(rowVisitor)
}

var urlsList1 []string
var urlsList2 []string

//var urlsList2 []string
func Empty(n int, m int) (empty []string) {
	f, err := excelize.OpenFile("result.xlsx")

	if err != nil {
		log.Fatal(err)
	}
	for i := n; i < m; i++ {
		cn := "B" + strconv.Itoa(i)
		kn := "A" + strconv.Itoa(i)

		c, err := f.GetCellValue("Sheet1", cn)
		k, err := f.GetCellValue("Sheet1", kn)

		if err != nil {
			log.Fatal(err)
		}
		if c == "[]" {
			empty = append(empty, k)

		}
	}
	fmt.Println(len(empty))
	return
}

func main() {
	start := time.Now()
	var url string

	rowStuff()
	for i := 1; i < len(words); i++ {

		word := strings.Split(words[i], " ")
		url = "https://www.bing.com/search?q="
		for j := 0; j < len(word); j++ {
			url = url + word[j] + "+"

		}
		url1 := url + "&first=10"
		//url2 := url + "&first=20"

		urlsList1 = append(urlsList1, url1)
		//urlsList2 = append(urlsList2, url2)

		url = ""
	}
	getURLDataWithRetries(urlsList1)

	f := excelize.NewFile()
	// Create a new sheet.
	index := f.NewSheet("Sheet1")
	// Set value of a cell.
	f.SetCellValue("Sheet1", "A1", "keyword")

	for i := 2; i < len(keywords); i++ {
		t := strconv.Itoa(i)
		cell := "A" + t
		f.SetCellValue("Sheet1", cell, keywords[i])

	}

	for i := 0; i < len(keywords)-2; i++ {

		var lines []string
		t := strconv.Itoa(i) + ".txt"
		n := strconv.Itoa(i + 2)
		file2, err := os.OpenFile(t, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

		if err != nil {
			log.Fatalf("failed creating file: %s", err)
		}
		file2.Close()
		file, err := os.Open(t)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		// optionally, resize scanner's capacity for lines over 64K, see next example
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
		cell := "B" + n
		f.SetCellValue("Sheet1", cell, lines)

	}
	// Set active sheet of the workbook.
	f.SetActiveSheet(index)
	// Save spreadsheet by the given path.
	if err := f.SaveAs("result.xlsx"); err != nil {
		fmt.Println(err)
	}

	elapsed := time.Since(start)
	log.Printf("Binomial took %s", elapsed)

	remove_all()
}

func remove_all() {
	for i := 0; i < 1000; i++ {
		e := os.Remove(strconv.Itoa(i) + ".txt")
		if e != nil {
			log.Fatal(e)
		}
	}
}
