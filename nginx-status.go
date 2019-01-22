package main

import (
	"bufio"
	"encoding/json"
	"os"
	"fmt"
	"net/http"
	"time"
	"flag"

	"github.com/fatih/color"
)

const DEFAULT_STATUS_URL = "http://localhost:80/nginx/check?format=json"

type NginxUpstreamCheckData struct {
	Servers struct {
			Total      uint64                     `json:"total"`
			Generation uint64                     `json:"generation"`
			Server     []NginxUpstreamCheckServer `json:"server"`
		} `json:"servers"`
}

type NginxUpstreamCheckServer struct {
	Index    uint64 `json:"index"`
	Upstream string `json:"upstream"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	Rise     uint64 `json:"rise"`
	Fall     uint64 `json:"fall"`
	Type     string `json:"type"`
	Port     uint64 `json:"port"`
}

func (server *NginxUpstreamCheckServer) getColorStatus() string {
	if server.Status == "up" {
		return color.GreenString(server.Status)
	} else if server.Status == "down" {
		return color.RedString(server.Status)
	} else {
		return color.BlueString(server.Status)
	}
}

func (server *NginxUpstreamCheckServer) getFullName() string {
	return fmt.Sprintf("%s://%s", server.Type, server.Name)
}

func (server *NginxUpstreamCheckServer) getRiseFallPort() string {
	if server.Port == 0 {
		return fmt.Sprintf("(r:%d,f:%d)", server.Rise, server.Fall)
	} else {
		return fmt.Sprintf("(r:%d,f:%d,c:%d)", server.Rise, server.Fall, server.Port)
	}
}

func (server *NginxUpstreamCheckServer) print() {
	fmt.Fprintf(
		color.Output, "[%s] %s - %s %s\n",
		server.getColorStatus(),
		server.Upstream,
		server.getFullName(),
		server.getRiseFallPort(),
	)
}

// -----

func parseFlags() string {
	var statusURL string
	flag.StringVar(&statusURL, "url", DEFAULT_STATUS_URL, "Nginx Upstream Checker status page URL")
	flag.Parse()
	return statusURL
}

func getRequest(url string) (*http.Request, error) {
	request, err := http.NewRequest("GET", url, nil)
        if err != nil {
		return nil, err
        }
        request.Header.Set("User-Agent", "Nginx-Status")
	return request, nil
}

func getData(url string) (*bufio.Reader, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	request, err := getRequest(url)

	if err != nil {
		return nil, err
	}

	response, err := client.Do(request)

	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP response code is %s", response.Status)
	}

	return bufio.NewReader(response.Body), nil
}

func main() {
	statusURL := parseFlags()

	buffer, err := getData(statusURL)

	if err != nil {
		fmt.Fprintf(color.Output, "%s %s\n", color.RedString("HTTP request have failed:"), err)
		os.Exit(1)
	}

	decoder := json.NewDecoder(buffer)
	checkData := &NginxUpstreamCheckData{}
	err = decoder.Decode(checkData)
	if err != nil {
		fmt.Fprintf(color.Output, "%s %s\n", color.RedString("JSON parsing have failed:"), err)
		os.Exit(1)
	}
	var previous_upstream string
	for _, server := range checkData.Servers.Server {
		if previous_upstream != "" && previous_upstream != server.Upstream {
			fmt.Println("")
		}
		server.print()
		previous_upstream = server.Upstream
	}
}
