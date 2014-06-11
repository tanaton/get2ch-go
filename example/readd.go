package main

import (
	"../"
	"bufio"
	"bytes"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"
)

type Nich struct {
	server string
	board  string
	thread string
}

const (
	GO_THREAD_SLEEP_TIME = 2 * time.Second
	GO_BOARD_SLEEP_TIME  = 4 * time.Second
)

var g_reg_bbs = regexp.MustCompile(`(.+\.2ch\.net|.+\.bbspink\.com)/(.+)<>`)
var g_reg_dat = regexp.MustCompile(`^([0-9]{9,10})\.dat<>`)
var g_reg_res = regexp.MustCompile(` \(([0-9]{1,4})\)$`)
var g_cache = get2ch.NewFileCache(`/2ch/dat`)
var gLogger = log.New(os.Stdout, "", log.LstdFlags)
var g_filter map[string]struct{} = map[string]struct{}{
	"epg.2ch.net":          struct{}{},
	"headline.2ch.net":     struct{}{},
	"headline.bbspink.com": struct{}{},
	"ipv6.2ch.net":         struct{}{},
}

func main() {
	// get2ch開始
	get2ch.Start(g_cache, nil)
	nsl := getServer()
	sl := nsl
	// クローラーの立ち上げ
	killch := startCrawler(sl)

	tick := time.Tick(time.Minute * 10)
	for _ = range tick {
		// 10分毎に板一覧を更新
		gLogger.Printf("Update server list\n")
		// 更新
		nsl = getServer()

		var flag bool
		if len(sl) != len(nsl) {
			// 鯖が増減した
			flag = true
		} else {
			for key, _ := range nsl {
				if _, ok := sl[key]; !ok {
					flag = true
					break
				}
			}
		}

		if flag {
			// 今のクローラーを殺す
			close(killch)
			// 鯖を更新
			sl = nsl
			// 新クローラーの立ち上げ
			killch = startCrawler(sl)
		}
	}
}

func startCrawler(sl map[string][]Nich) (killch chan struct{}) {
	// 新クローラー立ち上げ
	killch = make(chan struct{})
	for key, it := range sl {
		gLogger.Printf("Server:%s, Board_len:%d\n", key, len(it))
		go mainThread(key, it, killch)
		time.Sleep(GO_THREAD_SLEEP_TIME)
	}
	return
}

func checkOpen(ch <-chan struct{}) bool {
	select {
	case <-ch:
		// chanがクローズされると即座にゼロ値が返ることを利用
		return false
	default:
		break
	}
	return true
}

func mainThread(key string, bl []Nich, killch chan struct{}) {
	for {
		for _, nich := range bl {
			// 板の取得
			tl := getBoard(nich)
			if tl != nil && len(tl) > 0 {
				// スレッドの取得
				getThread(tl, nich.board, killch)
			}
			if checkOpen(killch) == false {
				// 緊急停止
				break
			}
			// 少し待機
			time.Sleep(GO_BOARD_SLEEP_TIME)
		}
		if checkOpen(killch) == false {
			// 緊急停止
			break
		}
		// 少し待機
		time.Sleep(GO_BOARD_SLEEP_TIME)
	}
}

func getServer() map[string][]Nich {
	var nich Nich
	get, _ := get2ch.NewGet2ch("", "")
	sl := make(map[string][]Nich, 16)
	// 更新時間を取得しない
	data := get.GetBBSmenu(false)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		if d := g_reg_bbs.FindStringSubmatch(scanner.Text()); len(d) > 0 {
			nich.server = d[1]
			nich.board = d[2]
			if _, ok := g_filter[nich.server]; ok {
				continue
			}
			nl, ok := sl[nich.server]
			if !ok {
				nl = make([]Nich, 0, 32)
			}
			sl[nich.server] = append(nl, nich)
		}
	}
	// 余分な領域を削る
	for board, it := range sl {
		l := len(it)
		sl[board] = it[:l:l]
	}
	return sl
}

func getBoard(nich Nich) []Nich {
	h := threadResList(nich)
	get, _ := get2ch.NewGet2ch(nich.board, "")
	data, err := get.GetData()
	if err != nil {
		gLogger.Printf(err.Error() + "\n")
		return nil
	}
	code := get.GetHttpCode()
	gLogger.Printf("%d %s/%s\n", code, nich.server, nich.board)
	if code != 200 {
		return nil
	}

	var n Nich
	vect := make([]Nich, 0, 32)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		it := scanner.Text()
		if da := g_reg_dat.FindStringSubmatch(it); da != nil {
			if d := g_reg_res.FindStringSubmatch(it); d != nil {
				n.server = nich.server
				n.board = nich.board
				n.thread = da[1]
				if m, ok := h[da[1]]; ok {
					if j, err := strconv.Atoi(d[1]); err == nil && m != j {
						vect = append(vect, n)
					}
				} else {
					vect = append(vect, n)
				}
			}
		}
	}
	l := len(vect)
	return vect[:l:l]
}

func threadResList(nich Nich) map[string]int {
	data, err := g_cache.GetData(nich.server, nich.board, "")
	h := make(map[string]int, 1024)
	if err != nil {
		return h
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		it := scanner.Text()
		if da := g_reg_dat.FindStringSubmatch(it); da != nil {
			if d := g_reg_res.FindStringSubmatch(it); d != nil {
				m, _ := strconv.Atoi(d[1])
				h[da[1]] = m
			}
		}
	}
	return h
}

func getThread(tl []Nich, board string, killch chan struct{}) {
	for _, nich := range tl {
		get, _ := get2ch.NewGet2ch(nich.board, nich.thread)
		_, err := get.GetData()
		if err != nil {
			gLogger.Println(err)
			gLogger.Printf("%s/%s/%s\n", nich.server, nich.board, nich.thread)
		} else {
			code := get.GetHttpCode()
			gLogger.Printf("%d %s/%s/%s\n", code, nich.server, nich.board, nich.thread)
		}
		if checkOpen(killch) == false {
			// 緊急停止
			break
		}
	}
}
