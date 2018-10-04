package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

const (
	inputStdin = "--"
	inputXSel  = "s"
	apiURL     = "https://translate.googleapis.com/translate_a/single" +
		"?client=gtx&sl=%s&tl=%s&dt=t&dt=bd&q=%s"
)

func main() {
	// checkDependencies()

	sourceLang := flag.String("s", "en", "Source language")
	targetLang := flag.String("t", "ru", "Target language")
	notifySend := flag.Bool("n", false, "Creates a notification (notify-send)")
	input := flag.String("i", "", fmt.Sprintf(`It means to read text:
	  %s   - from standart input
	  %s    - X server primary selection
	  text - as is`, inputStdin, inputXSel))

	flag.Parse()
	if *sourceLang == "" || *targetLang == "" || *input == "" {
		flag.Usage()
		os.Exit(1)
	}

	text, err := getText(*input)
	if err != nil {
		log.Fatal(err)
	}

	if text, err = getTranslation(*sourceLang, *targetLang, text); err != nil {
		log.Fatal(err)
	}

	if *notifySend {
		title := strings.Title(fmt.Sprintf("%s > %s", *sourceLang, *targetLang))
		err = exec.Command("notify-send", title, text).Run()
	} else {
		_, err = fmt.Println(text)
	}

	if err != nil {
		log.Fatal(err)
	}
}

func getText(in string) (res string, err error) {
	var tmp []byte
	switch {
	case in == inputStdin:
		tmp, err = ioutil.ReadAll(os.Stdin)
	case in == inputXSel:
		tmp, err = exec.Command("xsel", "--output", "--primary").Output()
	default:
		return in, err
	}
	res = string(tmp)
	return
}

func getTranslation(source, target, text string) (res string, err error) {
	uri := fmt.Sprintf(apiURL, source, target, url.QueryEscape(text))

	var req *http.Request
	if req, err = http.NewRequest("GET", uri, nil); err != nil {
		return
	}
	req.Header.Add("User-Agent", "")

	hc := new(http.Client)
	var resp *http.Response
	if resp, err = hc.Do(req); err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		err = errors.New("Google Translate: " + resp.Status)
		return
	}

	var body []byte
	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		return
	}

	var data []interface{}
	if err = json.Unmarshal(body, &data); err != nil {
		return
	}

	res = getTextTranslation(data)
	res += getWordTranslation(data)

	return
}

func getTextTranslation(data []interface{}) (res string) {
	for _, v := range data[:1] {
		for _, v := range v.([]interface{}) {
			res += v.([]interface{})[0].(string)
		}
	}
	return
}

func getWordTranslation(data []interface{}) (res string) {
	if data[1] == nil {
		return
	}

	for _, v := range data[1:2][0].([]interface{}) {
		res += "\n" + v.([]interface{})[0].(string) + ": "
		for _, v := range v.([]interface{})[1].([]interface{}) {
			res += v.(string) + ", "
		}
		res = strings.TrimRight(res, ", ")
	}
	res = strings.ToLower(res)

	return
}

func checkDependencies() {
	if _, err := exec.LookPath("xsel"); err != nil {
		log.Fatal(err)
	}
	if _, err := exec.LookPath("notify-send"); err != nil {
		log.Fatal(err)
	}
}
