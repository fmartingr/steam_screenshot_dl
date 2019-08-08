package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

/*
File list: https://steamcommunity.com/id/pyronhell/screenshots/screenshots
Body: appid=0&p=1&privacy=14&content=1&browsefilter=myfiles&sort=newestfirst&view=imagewall
                ^ Page number
Look for: OnScreenshotClicked( 427358024 );

File details: https://steamcommunity.com/sharedfiles/filedetails/?id=824709761
Look for:
	Image: <a href="https://steamuserimages-a.akamaihd.net/ugc/96097735799791630/2AA984CBC5D52A6105D76844193E3130B572F970/" target="_blank">
	AppName: <a href=\"https://steamcommunity.com/app/270/">AppName</a>
*/

// FileURL stores the app name and the screenshot URL to easily save the files
type FileURL struct {
	app string
	url string
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func getFileIDs(username string, page int) []string {
	fileIDs := []string{}
	fileRegex, _ := regexp.Compile("OnScreenshotClicked\\(\\s([\\d]+)\\s\\)")

	for {
		fmt.Printf("- Getting files from page %d\n", page)
		requestURL := fmt.Sprintf("https://steamcommunity.com/id/%s/screenshots/screenshots", username)
		response, err := http.PostForm(requestURL, url.Values{"appid": {"0"}, "p": {fmt.Sprintf("%d", page)}, "content": {"1"}, "view": {"imagewall"}, "sort": {"newestfirst"}, "privacy": {"14"}, "browsefilter": {"myfiles"}})
		check(err)

		defer response.Body.Close()
		body, err := ioutil.ReadAll(response.Body)
		check(err)

		matches := fileRegex.FindAllStringSubmatch(string(body), -1)
		for _, match := range matches {
			fileIDs = append(fileIDs, match[1])
		}

		if strings.Contains(string(body), "EndOfInfiniteContent") {
			break
		}
		page++
	}

	return fileIDs
}

func getFiles(fileIDs []string) []FileURL {
	files := []FileURL{}
	regexURL, _ := regexp.Compile("<a href=\"(.*akamaihd.*)\" target=\"_blank\">")
	// regexAppName, _ := regexp.Compile("<div class=\"apphub_AppName ellipsis\">(.*)</div>")
	regexAppName, _ := regexp.Compile("<a href=\"https://steamcommunity.com/app/[\\d]+\">(.*)</a>")

	for _, fileID := range fileIDs {
		fmt.Printf("- %s\n", fileID)

		requestURL := fmt.Sprintf("https://steamcommunity.com/sharedfiles/filedetails/?id=%s", fileID)
		response, err := http.Get(requestURL)
		check(err)

		defer response.Body.Close()
		body, err := ioutil.ReadAll(response.Body)
		check(err)

		fileURL := regexURL.FindStringSubmatch(string(body))[1]
		appName := regexAppName.FindStringSubmatch(string(body))[1]

		files = append(files, FileURL{app: appName, url: fileURL})
	}
	return files
}

func downloadFiles(files []FileURL, dest string) {
	for _, file := range files {
		os.MkdirAll(fmt.Sprintf("%s/%s", dest, file.app), os.ModePerm)

		response, err := http.Get(file.url)
		check(err)

		fmt.Printf("- %s", response.Header["Content-Disposition"][0])

		contentDisposition := strings.Split(response.Header["Content-Disposition"][0], "_")
		date := contentDisposition[2]
		number, err := strconv.ParseInt(strings.Split(contentDisposition[3], ".")[0], 10, 64)
		check(err)
		extension := strings.Trim(strings.Split(contentDisposition[3], ".")[1], ";\"")

		creationTime, err := time.Parse("20060102150405", date)
		if err != nil {
			// Old filename
			creationTime, err = time.Parse("2006-01-02", date)
			check(err)
			creationTime = creationTime.Add(time.Duration(number * 1e09))
		}
		destination := fmt.Sprintf("%s/%s/%s.%s", dest, file.app, creationTime.Format("2006-01-02_15-04-05"), extension)

		body, err := ioutil.ReadAll(response.Body)
		check(err)

		err = ioutil.WriteFile(destination, body, 0644)
		check(err)

		os.Chtimes(destination, creationTime, creationTime)
	}
}

func main() {
	arguments := os.Args[1:]

	if len(arguments) != 2 {
		fmt.Println("Please enter your username and destination path")
		os.Exit(1)
	}

	destPath := arguments[1]
	username := arguments[0]

	fmt.Println("=> Downloading screenshots for " + username)

	fileIDs := getFileIDs(username, 1)

	fmt.Println("=> Getting file list")
	files := getFiles(fileIDs)

	fmt.Println("=> Downloading files")
	downloadFiles(files, destPath)
}
