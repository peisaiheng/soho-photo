package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	SohoProd                  = "https://app.sohoapp.com/api"
	SohoStaging               = "https://staging.sohoapp.com/api"
	SohoManageEndpoint        = "/manage/property_listings" // required: listing_state
	SohoVerificationsProperty = "/verifications/property"
)

var Token string
var ApiPath string

func main() {

	ApiPath = SohoProd

	//open csv file from args
	fileName := os.Args[1]
	csvFile, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer csvFile.Close()
	r := csv.NewReader(bufio.NewReader(csvFile))

	//get each record and update photos for each listing.
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		user_id, property_id, listing_id, state, Token := record[0], record[1], record[2], record[3], record[4]
		fmt.Println(user_id, property_id, listing_id, state, Token)

		// Find the directory (photos of the listing should be in the folder named by property id)

		dirname := `./temp` + property_id + `/` // directory of photos for a single listing

		list, err := ReadDir(dirname)
		if err != nil {
			fmt.Println(err)
		}

		photoMap := make(map[string]io.Reader)
		// Range photos array and post photos & Prepare the reader instances to encode
		for index, file := range list {
			if strings.Contains(file.Name(), `.jpg`) {
				photoPath := dirname + file.Name()
				fmt.Println(photoPath)
				photoMap[`property_attributes[property_photos_attributes][`+strconv.Itoa(index)+`][image]`] = mustOpen(photoPath)
				photoMap[`property_attributes[property_photos_attributes][`+strconv.Itoa(index)+`][display_order]`] = strings.NewReader(strconv.Itoa(index))
			}
		}

		photoMap[`listing_type`] = strings.NewReader(state)

		// Upload photos
		fmt.Println(ApiPath + SohoManageEndpoint + "/" + listing_id)
		fmt.Println(photoMap)
		//err = Upload(ApiPath+SohoManageEndpoint+"/"+listing_id, photoMap)
		//if err != nil {
		//	fmt.Println(err)
		//}
	}
}

// ReadDir reads the directory named by dirname and returns
// a list of directory entries sorted by filename.
func ReadDir(dirname string) ([]os.FileInfo, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	list, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name() < list[j].Name() })
	return list, nil
}

func Upload(url string, values map[string]io.Reader) (err error) {
	// Prepare a form that you will submit to that URL.
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	for key, r := range values {
		var fw io.Writer
		if x, ok := r.(io.Closer); ok {
			defer x.Close()
		}
		// Add an image file
		if x, ok := r.(*os.File); ok {
			if fw, err = w.CreateFormFile(key, x.Name()); err != nil {
				return
			}
		} else {
			// Add other fields
			if fw, err = w.CreateFormField(key); err != nil {
				return
			}
		}
		if _, err = io.Copy(fw, r); err != nil {
			return err
		}

	}
	// Don't forget to close the multipart writer.
	// If you don't close it, your request will be missing the terminating boundary.
	w.Close()

	// Now that you have a form, you can submit it to your handler.
	req, err := http.NewRequest(http.MethodPut, url, &b)
	if err != nil {
		return
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Add("Authorization", Token)

	// Submit the request
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return
	}
	res.Body.Close()

	// Check the response
	if res.StatusCode != http.StatusOK {
		fmt.Printf("status: %s \n", res.Status)
	}
	return
}

func mustOpen(f string) *os.File {
	r, err := os.Open(f)
	if err != nil {
		fmt.Println("mustOpen Func Error: ", err)
	}
	return r
}
