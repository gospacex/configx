package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("usage: upload <dataId> <group> <file> [namespace]")
		return
	}
	dataID := os.Args[1]
	group := os.Args[2]
	filePath := os.Args[3]
	namespace := ""
	if len(os.Args) > 4 {
		namespace = os.Args[4]
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println("read error:", err)
		return
	}

	data := url.Values{}
	data.Set("dataId", dataID)
	data.Set("group", group)
	data.Set("type", "yaml")
	data.Set("content", string(content))
	if namespace != "" {
		data.Set("tenant", namespace)
	}

	resp, err := http.PostForm("http://localhost:8848/nacos/v1/cs/configs", data)
	if err != nil {
		fmt.Println("upload error:", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Upload:", string(body))

	// verify
	reqURL := "http://localhost:8848/nacos/v1/cs/configs?dataId=" + dataID + "&group=" + group
	if namespace != "" {
		reqURL += "&tenant=" + namespace
	}
	resp2, err := http.Get(reqURL)
	if err != nil {
		fmt.Println("verify error:", err)
		return
	}
	defer resp2.Body.Close()
	body2, _ := io.ReadAll(resp2.Body)
	if len(body2) > 100 {
		fmt.Printf("Verify OK, length=%d, first 80 chars: %s\n", len(body2), string(body2[:80]))
	} else {
		fmt.Println("Verify:", string(body2))
	}
}
