package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"testing"

	"github.com/k0kubun/pp"
	"github.com/temoto/robotstxt"
)

func TestUrl(t *testing.T) {
	urlStr := "https://www.jstage.jst.go.jp/test/browse/jspa1962/-char/ja/"
	u, _ := url.Parse(urlStr)
	var input string
	const version = "0.01"
	var userAgent = fmt.Sprintf("JstagoClient/%s (%s)", version, runtime.Version())
	pp.Println(u.RequestURI())
	pp.Println(u.String())
	pp.Println(u.Scheme + "://" + u.Host + "/robots.txt")
	robotsmap := make(map[string]*robotstxt.RobotsData, 0)
	// fmt.Println("走査起点となるURLを入力してください。(e.g. https://www.jstage.jst.go.jp/browse/jjoes)")
	// fmt.Scanln(&input)
	// urlStr := input
	u, err := url.Parse(urlStr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// robots.txt 取得
	req, err := http.NewRequest("GET", u.Scheme+"://"+u.Host+"/robots.txt", nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		fmt.Println(err)
		return
	}
	robots, err := robotstxt.FromResponse(resp)
	robotsmap[u.Host] = robots
	pp.Println(u.RequestURI(), robots.TestAgent(u.RequestURI(), userAgent))
	if !robots.TestAgent(u.RequestURI(), userAgent) {
		pp.Println(u.String(), "はrobots.txt によってサーバーから禁止されています。実行しますか？[y/n]")
		fmt.Scanln(&input)
		if input != "y" {
			return
		}
	}
}

// func TestCrawl(t *testing.T) {
// 	url := "https://www.jstage.jst.go.jp/browse/jspa1962/-char/ja/"
// 	resp, err := http.Get(url)
// 	if err != nil {
// 		fmt.Println(err)
// 		return
// 	}
// 	doc, err := goquery.NewDocumentFromResponse(resp)
// 	if err != nil {
// 		fmt.Println(err)
// 		return
// 	}
// 	// book title
// 	bt := strings.TrimSpace(doc.Find("h1.mod-page-heading").Text())
// 	pchan := make(chan *PdfMeta, 2)
// 	var skip bool
// 	go func() {
// 		for {
// 			p := <-pchan
// 			if skip {
// 				pp.Println("[スキップ] ファイル名:" + p.SaveTo)
// 				skip = false
// 				return
// 			}
// 			pp.Println("[ダウンロード] ファイル名:" + p.SaveTo)

// 			response, err := http.Get(p.Url)
// 			if err != nil {
// 				if response != nil {
// 					response.Body.Close()
// 				}
// 				fmt.Println(err)
// 				return
// 			}
// 			body, err := ioutil.ReadAll(response.Body)
// 			if err != nil {
// 				fmt.Println(err)
// 				os.Exit(1)
// 			}

// 			if err := ioutil.WriteFile(p.SaveTo, body, os.ModePerm); err != nil {
// 				switch e := err.(type) {
// 				case *os.PathError:
// 					d, _ := path.Split(e.Path)
// 					if err := os.MkdirAll(d, 0777); err != nil {
// 						fmt.Print(err)
// 						return
// 					}
// 					ioutil.WriteFile(p.SaveTo, body, os.ModePerm)
// 				default:
// 					fmt.Print("failed to write", err)
// 					return
// 				}
// 			}
// 			fmt.Println("ok", p.Url, "==>>", p.SaveTo)

// 			rand.Seed(time.Now().UnixNano())
// 			r := rand.Intn(5) + 1
// 			pp.Println("[待機]" + strconv.Itoa(r) + "秒...\n")
// 			time.Sleep(time.Second * time.Duration(r))
// 		}
// 	}()

// 	doc.Find("div.mod-item").Each(func(_ int, s *goquery.Selection) {
// 		an := strings.TrimSpace(s.Find("h3.mod-item-heading > a").Text())
// 		pa := strings.TrimSpace(s.Find("div.mod-item-pagearea").Text())
// 		a, ok := s.Find("ul > li.icon-pdf_key > a").Attr("href")
// 		if !ok {
// 			pp.Println("no href")
// 			return
// 		}
// 		meta := s.Find("div.mod-item-meta > p").First().Text()
// 		p := &PdfMeta{
// 			Author:   meta,
// 			Title:    an,
// 			Journal:  bt,
// 			Url:      baseurl + a,
// 			PageArea: pa,
// 		}
// 		tmpl := "{{.Journal}}/{{.PageArea}}-{{.Author}}-{{.Title}}"
// 		tpl := template.Must(template.New("t").Parse(tmpl))
// 		var doc bytes.Buffer
// 		if err := tpl.Execute(&doc, p); err != nil {
// 			fmt.Println(err)
// 			ps := strings.Split(p.Url, "/")
// 			p.SaveTo = ps[len(ps)-2] + ext
// 		} else {
// 			p.SaveTo = strings.TrimSpace(doc.String()) + ext
// 		}
// 		pp.Println(p)
// 		pchan <- p
// 	})

// }
