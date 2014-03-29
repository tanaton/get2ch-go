package main

import (
	"log"
	"regexp"
	"strconv"
	"bytes"
	"bufio"
	"time"
	"../"
)

type Nich struct {
	server		string
	board		string
	thread		string
}

const GO_THREAD_SLEEP_TIME = 2 * time.Second
const THREAD_SLEEP_TIME = 4 * time.Second

var g_reg_bbs = regexp.MustCompile("(.+\\.2ch\\.net|.+\\.bbspink\\.com)/(.+)<>")
var g_reg_dat = regexp.MustCompile("^(.+)\\.dat<>")
var g_reg_res = regexp.MustCompile(" [(]([0-9]+)[)]$")
var g_cache = get2ch.NewFileCache("/2ch/dat")

var g_filter map[string]bool = map[string]bool{
	"ipv6.2ch.net"			: true,
	"headline.2ch.net"		: true,
	"headline.bbspink.com"	: true,
}

func main() {
	var sl, nsl map[string][]Nich
	var key string
	// get2ch開始
	get2ch.Start(g_cache, nil)
	sync := make(chan string)
	// メイン処理
	sl = getServer()
	for key = range sl {
		if h, ok := sl[key]; ok {
			go mainThread(key, h, sync)
			time.Sleep(GO_THREAD_SLEEP_TIME)
		}
	}
	for {
		// 処理を止める
		key = <-sync
		nsl = getServer()
		for k := range nsl {
			if _, ok := sl[k]; !ok {
				go mainThread(k, nsl[k], sync)
				time.Sleep(GO_THREAD_SLEEP_TIME)
			}
		}
		if h, ok := nsl[key]; ok {
			go mainThread(key, h, sync)
			time.Sleep(GO_THREAD_SLEEP_TIME)
		}
		sl = nsl
	}
}

func mainThread(key string, bl []Nich, sync chan string) {
	for _, nich := range bl {
		// 板の取得
		tl := getBoard(nich)
		if tl != nil && len(tl) > 0 {
			// スレッドの取得
			getThread(tl)
		}
		// 止める
		time.Sleep(THREAD_SLEEP_TIME)
	}
	sync <- key
}

func getServer() map[string][]Nich {
	var nich Nich
	get := get2ch.NewGet2ch("", "")
	sl := make(map[string][]Nich)
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
			if h, ok := sl[nich.server]; ok {
				sl[nich.server] = append(h, nich)
			} else {
				sl[nich.server] = []Nich{nich}
			}
		}
	}
	return sl
}

func getBoard(nich Nich) []Nich {
	get := get2ch.NewGet2ch(nich.board, "")
	h := threadResList(nich)
	data, err := get.GetData()
	if err != nil {
		log.Printf(err.Error() + "\n")
		return nil
	}
	get.GetBoardName()
	vect := make([]Nich, 0, 128)
	var n Nich
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		it := scanner.Text()
		if da := g_reg_dat.FindStringSubmatch(it); len(da) > 0 {
			if d := g_reg_res.FindStringSubmatch(it); len(d) > 0 {
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
	return vect
}

func threadResList(nich Nich) map[string]int {
	data, err := g_cache.GetData(nich.server, nich.board, "")
	h := make(map[string]int)
	if err != nil { return h }
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		it := scanner.Text()
		if da := g_reg_dat.FindStringSubmatch(it); len(da) > 0 {
			if d := g_reg_res.FindStringSubmatch(it); len(d) > 0 {
				m, _ := strconv.Atoi(d[1])
				h[da[1]] = m
			}
		}
	}
	return h
}

func getThread(tl []Nich) {
	for _, nich := range tl {
		get := get2ch.NewGet2ch(nich.board, nich.thread)
		_, err := get.GetData()
		if err != nil {
			log.Printf(err.Error() + "\n")
			log.Printf("%s/%s/%s\n", nich.server, nich.board, nich.thread)
		} else {
			log.Printf("%d OK %s/%s/%s\n", get.GetHttpCode(), nich.server, nich.board, nich.thread)
		}
		// 4秒止める
		time.Sleep(THREAD_SLEEP_TIME)
	}
}

