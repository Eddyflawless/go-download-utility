package main

import (
	"fmt"
	"os"
	"net/http"
	"strconv"
	"log"
	"io/ioutil"
	"errors"
	"time"
	"mime"
	"sync"
)



const (
	tmpDir 		string =  	"tmp"
	exportDir 	string = 	"exports"
)


type Download struct { 
	Url string
	TargetPath string
	TotalSections int
}

func (d Download) Do() error {

	fmt.Println("Making a connection...")

	r, err := d.getNewRequest("HEAD") //HEAD allows us to get details of the file

	if err != nil { 
		return err
	}

	resp, err := http.DefaultClient.Do(r)

	if err != nil { 
		return err 
	}

	fmt.Printf("Got %v \n",resp.StatusCode)

	if resp.StatusCode > 299 {
		return errors.New(fmt.Sprintf("Can't process, response is %v", resp.StatusCode))
	}

	size, err := strconv.Atoi(resp.Header.Get("Content-Length"))

	if err != nil{
		return err
	}

	fmt.Printf("size is %v \n", size)

	var sections = make([][2]int, d.TotalSections)

	eachSize := size / d.TotalSections

	//example: if the file size is 20 bytes,  owur section will look like this
	//[[0,5],[6,10],[11,15],[16,20]]

	for i := range sections {

		if i == 0 {
			//starting byte of first section
			sections[i][0] = 0
		} else {
			//starting byte of other sections
			sections[i][0] = sections[i-1][1] + 1

		}

		if i < d.TotalSections - 1{
			//end byte of other sections
			sections[i][1] = sections[i][0] + eachSize
		} else {
			// ending byte of other sections
			sections[i][1] = size - 1
		}
	}

	fmt.Printf("sections %v \n", sections)

	var wg sync.WaitGroup
	for i, section := range sections {

		wg.Add(1)
		//store current value as they will be changing

		i := i
		s := section


		go func(){

			defer wg.Done()

			err = d.downloadSection(i,s)
	
			if err != nil {
				panic(err)
			}

		}()

	} 
	
	wg.Wait()

	err = d.mergeFiles(sections)

	if err != nil {
		return err
	}

	return nil

}

func (d Download) getNewRequest(method string) (*http.Request, error) {

	r, err := http.NewRequest(
		method, 
		d.Url, 
		nil,
	)

	if err != nil  {
		return nil, err
	}

	r.Header.Set("User-Agent", "Silly Download Manager ")

	return r, nil
}

func (d Download) downloadSection(i int, s [2]int) error {

	r, err := d.getNewRequest("GET")

	if err != nil {
		return err
	}

	r.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", s[0],s[1]) )

	resp, err := http.DefaultClient.Do(r)

	if err != nil {
		return err
	}

	fmt.Printf("Downloaded %v bytes  for section %v:%v \n", resp.Header.Get("Content-Length"), i, s)

	contentDisposition := resp.Header.Get("Content-Disposition")

	disposition, params, err := mime.ParseMediaType(contentDisposition)
	if err != nil {
		panic(err)
	}

	d.TargetPath = params["filename"]

	fmt.Println("Disposition is", disposition, "and filename is", d.TargetPath )

	b, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(fmt.Sprintf("%v/section-%v.tmp",tmpDir,i),b,os.ModePerm)

	if err != nil {
		return err
	}

	return nil
}

func (d Download) mergeFiles(sections [][2]int) error {

	fmt.Printf("==> section %v", sections)

	downloadDir := fmt.Sprintf("%v/%v",exportDir,d.TargetPath)

	f, err := os.OpenFile(downloadDir, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		return err
	}

	defer f.Close()

	for i := range sections {

		b, err := ioutil.ReadFile(fmt.Sprintf("%v/section-%v.tmp",tmpDir,i))

		if err != nil {
			return err
		}

		n, err := f.Write(b)

		if err != nil {
			return err
		}

		fmt.Printf("Bytes %v merged \n", n)
	}

	return nil
}


func main(){


	if len(os.Args) == 1 {
		panic("Provide a target url")
	}

	start_time := time.Now()

	for _,a := range os.Args {

		fmt.Printf("arg %v \n",a )

		// code here
		
		var d Download
		d = Download{
			Url: "https://unsplash.com/photos/ZeEpOxzLrfg/download?ixid=MnwxMjA3fDB8MXxhbGx8Mnx8fHx8fDJ8fDE2NDYzMjMyNjA&force=true&w=2400",
			TargetPath: "final.jpg",
			TotalSections: 10,
		}
	
		err := d.Do()
	
		if err != nil {
	
			log.Fatalf("An error occurred while downloading the file: %s\n", err)
	
		}
	
		fmt.Printf("Download completed for %v in %v seconds\n", a ,time.Now().Sub(start_time).Seconds())
	}


	fmt.Printf("Total of %v download completed in %v seconds\n", len(os.Args) ,time.Now().Sub(start_time).Seconds())

}