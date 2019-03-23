package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/k0kubun/pp"
	"github.com/temoto/robotstxt"
)

const (
	baseurl = "https://www.jstage.jst.go.jp"
	ext     = ".pdf"
)

type PdfMeta struct {
	Author   string
	Title    string
	Journal  string
	Year     int
	PageArea string
	URL      *url.URL
	SaveTo   string
}

func main() {
	var input string
	//同時コネクト数 (jstageサーバーへの負担をかけすぎないようにしてください)
	connect := 2
	checkRobots := true
	// file name template
	tmpl := "{{.Journal}}/{{.PageArea}}-{{.Author}}-{{.Title}}"
	const version = "0.01"
	var userAgent = fmt.Sprintf("JstagoClient/%s (%s)", version, runtime.Version())

	robotsmap := make(map[string]*robotstxt.RobotsData, 0)
	fmt.Println("走査起点となるURLを入力してください。(e.g. https://www.jstage.jst.go.jp/browse/jjoes)")
	fmt.Scanln(&input)
	urlStr := input
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
	if checkRobots && !robots.TestAgent(u.RequestURI(), userAgent) {
		pp.Println(u.String(), "はrobots.txt によってサーバーから禁止されています。実行しますか？[y/n]")
		fmt.Scanln(&input)
		if input != "y" {
			pp.Println("キャンセルしました。")
			return
		}
	}

	req, err = http.NewRequest("GET", u.String(), nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		fmt.Println(err)
		return
	}

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		fmt.Print("url scrap failed", err)
		return
	}

	links := make([]string, 0)
	fmt.Println("巻号一覧を取得します...(検索語: <No., Vol.>)")

	// 雑
	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		if strings.Contains(s.Text(), "No.") || strings.Contains(s.Text(), "Vol.") {
			f, ok := s.Attr("href")
			if !ok {
				return
			}
			if f != "" {
				pp.Println(f)
				links = append(links, baseurl+f)
			}
		}
	})

	// 重複URL除去
	dlinks := make([]string, 0)
	encountered := map[string]bool{}
	for _, arg := range links {
		if arg == "" {
			continue
		}
		if !encountered[arg] {
			encountered[arg] = true
			dlinks = append(dlinks, arg)
		}
	}

	var auto, skip bool
	fmt.Println(strconv.Itoa(len(dlinks)) + "件のリンクを捕捉しました。これらを走査しますか？[y/n]")
	fmt.Scanln(&input)
	if !strings.HasPrefix(input, "y") {
		fmt.Println("終了します。")
		os.Exit(1)
	}

	fmt.Println("保存名を入力してください。/(スラッシュ)でフォルダ分けが出来ます。")
	fmt.Println("標準設定: " + tmpl + " で良いですか？[y/n]")
	fmt.Scanln(&input)
	if !strings.HasPrefix(input, "y") {
	tmploop:
		for {
			fmt.Println("標準設定の表記にならって入力してください")
			fmt.Scanln(&input)
			if !strings.ContainsAny(input, "{}") {
				fmt.Println("同一ファイルに上書きが繰り返されてしまいます。テンプレートには、{{}}が使われている必要があります。")
				continue
			}
			ptmpl, err := template.New("t").Parse(input)
			if err != nil {
				fmt.Println("不正なテンプレートです。", err)
				continue
			}
			// プレビュ-用dummy data
			dummy := &PdfMeta{
				Author: "某 太郎",
				Title: "論文の自動収集方法試論",
				Journal: "某学会誌",
				URL: &url.URL{
					Scheme: "https://dummy",
				},
				PageArea: "p.1_3-1_15",
			}
			var buf bytes.Buffer
			if err := ptmpl.Execute(&buf, dummy); err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Println("[preview] " + strings.TrimSpace(buf.String()) + ext)
			fmt.Println("これでよろしいですか？[y/n]")
			fmt.Scanln(&input)
			if !strings.HasPrefix(input, "y") {
				continue
			}
			tmpl = input
			break tmploop
		}
	}
	tpl, err := template.New("t").Parse(tmpl)

	fmt.Println("オートダウンロードをONにしますか？[y/n] 途中からでも[auto]でONにできます。")
	fmt.Scanln(&input)
	if !strings.HasPrefix(input, "y") {
		auto = true
	}
	pp.Println("AutoDownload:", auto)
	// message 応答部
	msg := make(chan string, 1)
	go func() {
		for {
			fmt.Scanln(&input)
			pp.Println("received: ", input)
			auto = false
			switch input {
			case "a", "auto":
				auto = true
				pp.Println("他のキーを押すと途中からでも[auto]をOFFにできます。")
			case "s", "skip":
				skip = true
			case "e", "end", "exit":
				msg <- "exit"
			}
			pp.Println("AutoDownload:", auto)
		}
	}()

	pchan := make(chan *PdfMeta, connect)
	// book title
	go func() {
		for {
			p := <-pchan
			if skip {
				pp.Println("[スキップ] ファイル名:" + p.SaveTo)
				skip = false
				continue
			}
			robots, ok := robotsmap[p.URL.Host]
			if !ok {
				fmt.Println("no robots history...")
				continue
			}

			if checkRobots && !robots.TestAgent(p.URL.RequestURI(), userAgent) {
				pp.Println(p.URL.String(), "はrobots.txt によってサーバーから禁止されています。実行しますか？[y/n]")
				fmt.Scanln(&input)
				if input != "y" {
					pp.Println("キャンセルしました。")
					continue
				}
			}
			pp.Println("[ダウンロード] ファイル名:" + p.SaveTo)
			pp.Println(p)
			req, err := http.NewRequest("GET", p.URL.String(), nil)
			if err != nil {
				fmt.Println(err)
				continue
			}
			req.Header.Set("User-Agent", userAgent)
			response, err := http.DefaultClient.Do(req)
			if err != nil {
				if response != nil {
					response.Body.Close()
				}
				fmt.Println(err)
				return
			}

			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			if err := ioutil.WriteFile(p.SaveTo, body, os.ModePerm); err != nil {
				switch e := err.(type) {
				case *os.PathError:
					d, _ := path.Split(e.Path)
					if err := os.MkdirAll(d, 0777); err != nil {
						fmt.Print(err)
						return
					}
					ioutil.WriteFile(p.SaveTo, body, os.ModePerm)
				default:
					fmt.Print("failed to write", err)
					return
				}
			}
			fmt.Println("ok", p.URL.String(), "==>>", p.SaveTo)

			rand.Seed(time.Now().UnixNano())
			r := rand.Intn(7) + 3
			pp.Println("[待機]" + strconv.Itoa(r) + "秒...\n")
			time.Sleep(time.Second * time.Duration(r))
		}
	}()

	for k, link := range dlinks {
		if !auto {
			fmt.Println(strconv.Itoa(k) + "件目のリンク(" + link + ")を自動検索しますか？ [y/n]")
			fmt.Scanln(&input)
		}
		if !auto && !strings.HasPrefix(input, "y") {
			fmt.Println("スキップします。")
			continue
		}
		u, err := url.Parse(link)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		robots, ok := robotsmap[u.Host]
		if !ok {
			// robots.txt 取得
			req, err := http.NewRequest("GET", u.Scheme+"://"+u.Host+"/robots.txt", nil)
			if err != nil {
				fmt.Println(err)
				return
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
			robots, err = robotstxt.FromResponse(resp)
			if err != nil {
				fmt.Println(err)
				return
			}
			robotsmap[u.Host] = robots
		}
		if checkRobots && !robots.TestAgent(u.RequestURI(), userAgent) {
			pp.Println(u.String(), "はrobots.txt によってサーバーから禁止されています。実行しますか？[y/n]")
			fmt.Scanln(&input)
			if input != "y" {
				pp.Println("キャンセルしました。")
				return
			}
		}

		req, err := http.NewRequest("GET", u.String(), nil)
		if err != nil {
			fmt.Println(err)
			return
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

		doc, err := goquery.NewDocumentFromResponse(resp)
		if err != nil {
			fmt.Print("url scrap failed", err)
			continue
		}
		// book title
		bt := strings.TrimSpace(doc.Find("h1.mod-page-heading").Text())

		doc.Find("div.mod-item").Each(func(_ int, s *goquery.Selection) {
			an := strings.TrimSpace(s.Find("h3.mod-item-heading > a").Text())
			pa := strings.TrimSpace(s.Find("div.mod-item-pagearea").Text())
			a, ok := s.Find("ul > li.icon-pdf_key > a").Attr("href")
			if !ok {
				pp.Println("no href")
				return
			}
			ref, err := url.Parse(a)
			if err != nil {
				fmt.Println(err)
				return
			}
			// 外部ページである。DLは控える。
			if ref.Host != u.Host {
				pp.Println(u.Host + "の外部ページです。>> " + ref.String())
				return
			}
			meta := s.Find("div.mod-item-meta > p").First().Text()
			u.Path = a
			p := &PdfMeta{
				Author:   meta,
				Title:    an,
				Journal:  bt,
				URL:      u,
				PageArea: pa,
			}

			var buf bytes.Buffer
			if err := tpl.Execute(&buf, p); err != nil {
				fmt.Println(err)
				ps := strings.Split(p.URL.String(), "/")
				p.SaveTo = ps[len(ps)-2] + ext
			} else {
				p.SaveTo = strings.TrimSpace(buf.String()) + ext
			}
			pchan <- p
		})

		fmt.Println("3秒間入力を待ち受けます。")
		select {
		case <-time.After(time.Second * 3):
			pp.Println("次のリンクを走査します。")
			continue
		case m := <-msg:
			switch m {
			case "exit":
				goto next
			}
		}
	next:
	}
	pp.Println("すべての作業が終わりました。")
}
