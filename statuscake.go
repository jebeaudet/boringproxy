package boringproxy

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

var ips []string = nil

func GetStatusCakeIps() ([]string, error) {
	var err error
	if ips == nil {
		err = initialize()
	}
	return ips, err
}

func initialize() error {
	err := updateStatusCakeIpsFromApi()
	go refreshIps()
	return err
}

func updateStatusCakeIpsFromApi() error {
	url := "https://app.statuscake.com/Workfloor/Locations.php?format=json"

	response, err := http.Get(url)
	if err != nil {
		log.Println("Failed to retrieve JSON:", err)
		return err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println("Failed to read response body:", err)
		return err
	}

	var jsonData map[string]map[string]interface{}
	err = json.Unmarshal(body, &jsonData)
	if err != nil {
		log.Println("Failed to parse JSON:", err)
		return err
	}

	ipList := make([]string, 0)
	for _, data := range jsonData {
		ip := data["ip"].(string)
		ipList = append(ipList, ip)
	}

	ips = ipList
	return nil
}

func refreshIps() {
	for {
		if ips == nil {
			time.Sleep(time.Minute)
		} else {
			time.Sleep(time.Hour)
		}
		updateStatusCakeIpsFromApi()
	}
}
