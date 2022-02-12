package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

type endPoint struct {
	name   string
	link   string
	subdir string
}

type ParsedJson map[string]listingInfo

type listingInfo struct {
	Id       int    `json:"Id"`
	Link     string `json:"Link"`
	MailSent bool   `json: "MailSent"`
}

func main() {
	port := goDotEnvVariable("$PORT")

	fmt.Println("HEROKU BRANCH")
	fmt.Println(port)
	for {
		callback()
		time.Sleep(5 * time.Minute)
	}
}

func acceptCookies(page *rod.Page) {
	page.MustWaitLoad().MustElement("[data-testid = 'cookie-policy-dialog-accept-button']").MustClick()
	fmt.Println("Accepted Cookies")
}

func login(page *rod.Page, username, password string) {

	fbUsername := goDotEnvVariable("USERNAME")
	fbPassword := goDotEnvVariable("PASSWORD")

	wait := page.WaitEvent(&proto.PageLoadEventFired{})
	page.MustWaitLoad().MustElement("[data-testid='royal_email']").MustInput(fbUsername)
	page.MustWaitLoad().MustElement("[data-testid='royal_pass']").MustInput(fbPassword)
	page.MustWaitLoad().MustElement("[data-testid='royal_login_button']").MustClick()

	wait()
	fmt.Println("Login Completed")
}

func goToMarketPlace(page *rod.Page) {
	//set max price for listings
	maxPrice := "350"

	page.MustNavigate("https://www.facebook.com/marketplace/category/propertyrentals?minPrice=10&maxPrice=" + maxPrice + "&exact=false")
	page.WaitEvent(&proto.PageLoadEventFired{})
	wait := page.MustWaitNavigation()

	wait()

	page.MustWaitLoad()
	var button proto.InputMouseButton
	err := page.MustElement("html").Click(button)
	if err != nil {
		fmt.Println("Error while clicking on html :", err)
	}

	for i := 1; i < 1500; i++ { // vale to sto 1500 gia na ta pairneis ola!!!
		page.MustEval(`window.scrollTo(0,document.body.scrollHeight);`)
		page.MustEval(`console.log("trexei")`)
		page.WaitIdle(1 * time.Second)
	}

	fmt.Println("Visited Marketplace")
}

func scanMarketPlaceListings(page *rod.Page, browser *rod.Browser, selector string) {
	fmt.Println("Scanning for Listings")

	page.MustWaitLoad()
	page.WaitEvent(&proto.PageLoadEventFired{})

	elements, err := page.Elements(selector)

	if err != nil {
		fmt.Println("ERROR while page.Elements", err)
	}

	fmt.Println("----------------elements----------------")
	fmt.Println(elements)
	fmt.Println("----------------selector----------------")
	fmt.Println(selector)

	// listings := []listingInfo{}

	listings := make(map[int]listingInfo)

	for _, el := range elements {

		hrefAttribute, _ := el.Attribute("href")
		hrefSplitArr := strings.Split(*hrefAttribute, "/")

		elementID, _ := strconv.Atoi(hrefSplitArr[3])
		elementLink := "https://www.facebook.com" + string(*hrefAttribute)

		listing := listingInfo{
			Id:       elementID,
			Link:     elementLink,
			MailSent: false,
		}

		listings[listing.Id] = listing

		// listings = append(listings, listing)
	}

	// check if directoy files exists, if not, make it
	makeDirIfNotExists("files", 0777)

	// check if fil alreadySent.json exists inside files folder, if not, make it
	makeFileIfNotExists("alreadySent.json", 0777)

	fmt.Println("current listings befere beeing passed in checkListingsFn()")
	fmt.Println(listings)

	checkListingsAndSendMails(listings)

}

func goDotEnvVariable(key string) string {

	// load .env file
	// err := godotenv.Load(".env")

	// if err != nil {
	// 	fmt.Println("Error loading .env file")
	// }

	value := os.Getenv(key)

	return value
}

func makeDirIfNotExists(dirname string, permissions int) {
	if _, err := os.Stat("files"); os.IsNotExist(err) {
		err := os.Mkdir("files", 0777)
		if err != nil {
			fmt.Println("ERROR: ", err)
		}
	}
}

func makeFileIfNotExists(filename string, permissions int) {
	if _, err := os.Stat("files/" + filename); errors.Is(err, os.ErrNotExist) {
		fmt.Println("ERROR: ", err)
		fmt.Println("Seems that files/alreadySent.json does not exist, will create it")
		os.Create("files/" + filename)
		ioutil.WriteFile("files/alreadySent.json", []byte("{}"), 0777)
	}
}

func checkListingsAndSendMails(currentListings map[int]listingInfo) {
	alreadySentListingsBytes, err := ioutil.ReadFile("files/alreadySent.json")

	if err != nil {
		fmt.Println("ERROR while reading files/alreadySent.json")
	}

	var alreadySentListings map[int]listingInfo

	json.Unmarshal(alreadySentListingsBytes, &alreadySentListings)

	// current listings => listings
	// fmt.Println("fetched Listings :")
	// fmt.Println(currentListings)
	// fmt.Println("already sent Listings :")
	// fmt.Println(alreadySentListings)
	// emailBody := "New Listings for : " + currentTime.Format("2006-01-02 15:04:05")

	emailBody := ""

	newEntriesExist := false

	fmt.Println("CURRENT LISTINGS")
	fmt.Println(currentListings)
	fmt.Println("-----------------")

	for key, listing := range currentListings {

		listingObj, _ := currentListings[key]

		if _, alreadyExists := alreadySentListings[key]; !alreadyExists {
			newEntriesExist = true
			// fmt.Println("Sending mail for ", key)

			emailBody += "\n"
			emailBody += "-------------------------------------"
			emailBody += "\n"
			emailBody += "Sending mail for " + strconv.Itoa(key)
			emailBody += "\n"
			emailBody += " " + listingObj.Link
			emailBody += "\n"
			emailBody += "-------------------------------------"
			emailBody += "\n"

			alreadySentListings[key] = listing
		}

	}

	if newEntriesExist {
		//send maila
		fmt.Println(emailBody)
		sendMail(emailBody)
	} else {
		fmt.Println("No new Listings")
	}

	data, _ := json.Marshal(alreadySentListings)

	// Update alreadySent.json with sent listings keys
	ioutil.WriteFile("files/alreadySent.json", data, 0777)

	fmt.Println("files checked")
}

func callback() {

	// parseFileAndSendMails()
	// os.Exit(111)

	fbUsername := goDotEnvVariable("USERNAME")
	fbPassword := goDotEnvVariable("PASSWORD")

	start := time.Now()

	urlMap := []endPoint{

		{
			name:   "facebook-home",
			link:   "https://el-gr.facebook.com/",
			subdir: "home",
		},
		// {
		// 	name :"google",
		// 	link: "https://www.google.com/",
		// 	subdir: "home",
		// },
		// {
		// 	name :"youtube",
		// 	link: "https://www.youtube.com/",
		// 	subdir: "home",
		// },
		// {
		// 	name :"roletraining",
		// 	link: "http://www.roletraining.gr/",
		// 	subdir: "home",
		// },
	}

	c := make(chan string)

	for _, item := range urlMap {

		go func(item endPoint, c chan string) {
			viewport := proto.EmulationSetDeviceMetricsOverride{
				Height: 1000,
			}

			fmt.Println("running for ", item)

			window := proto.BrowserBounds{
				Left:   0,
				Top:    0,
				Width:  1800,
				Height: 1000,
			}

			u := launcher.New().
				// Headless(true).
				NoSandbox(true).
				MustLaunch()

			browser := rod.New().ControlURL(u).MustConnect()
			page := browser.MustPage(item.link)

			page.SetViewport(&viewport)
			page.SetWindow(&window)

			// Step 1
			acceptCookies(page)

			// Step 2
			login(page, fbUsername, fbPassword)

			// Step 3
			goToMarketPlace(page)

			// step 4
			homeListingSelector := ".sonix8o1>span>div>div>a"
			scanMarketPlaceListings(page, browser, homeListingSelector)

			// Last step
			page.MustScreenshot("./tmp/" + item.name + "/" + item.subdir + ".png")
			fmt.Println("took a screnshot")

			c <- "Fetch for " + item.name + " ok!"

			browser.Close()
		}(item, c)

	}

	for i := 0; i < len(urlMap); i++ {
		fmt.Println(<-c)
	}

	elapsed := time.Since(start)
	log.Printf("Total Fetch took %s", elapsed)

}

func sendMail(mailContent string) {
	// Configuration
	currentTime := time.Now()
	emailHead := "New Listings for : " + currentTime.Format("2006-01-02 15:04:05")

	defaultMessage := goDotEnvVariable("DEFAULT_MSG")

	msg := []byte("To: spiropoulos94@gmail.com\r\n" +
		"Subject:" + emailHead + "\r\n" +
		"\r\n" + defaultMessage + "\r\n" +
		"\r\n" + mailContent + "\r\n")

	myMailAddress := goDotEnvVariable("TEST_MAIL")
	myMailPass := goDotEnvVariable("TEST_MAIL_PASSWORD")
	myMainMail := goDotEnvVariable("REAL_MAIL")

	from := myMailAddress
	password := myMailPass
	to := []string{myMainMail}
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	// Create authentication
	auth := smtp.PlainAuth("", from, password, smtpHost)

	// Send actual message
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, msg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("MAIL SENT SUCCESSFULLY")
}
