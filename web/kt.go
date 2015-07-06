package main

import (
	// log "code.google.com/p/log4go"
	myrpc "github.com/weisd/gopush-cluster/rpc"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// GetServer handle for server get for ktkt
func GetServerKtV2(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}
	params := r.URL.Query()
	key := params.Get("k")
	callback := params.Get("cb")
	protoStr := params.Get("p")
	res := map[string]interface{}{"ret": OK}
	defer retWrite(w, r, res, callback, time.Now())
	if key == "" {
		res["ret"] = ParamErr
		return
	}
	// Match a push-server with the value computed through ketama algorithm
	node := myrpc.GetComet(key)
	if node == nil {
		res["ret"] = NotFoundServer
		return
	}
	addrs, ret := getProtoAddr(node, protoStr)
	if ret != OK {
		res["ret"] = ret
		return
	}

	// weisd 处理内外网ip对应
	map_list := map[string]string{}
	for _, m := range Conf.IpMaps {
		mArr := strings.Split(m, ":")
		map_list[mArr[0]] = mArr[1]
	}

	fix_addrs := make([]string, 0)
	for _, addr := range addrs {
		sArr := strings.Split(addr, ":")
		ip := sArr[0]
		port := sArr[1]
		if outAddr, ok := map_list[ip]; ok {
			ip = outAddr
		}

		// 添加数组
		fix_addrs = append(fix_addrs, ip+":"+port)
	}
	// 随机
	randSource := rand.New(rand.NewSource(time.Now().UnixNano()))
	idx := randSource.Intn((len(fix_addrs) - 1))

	res["data"] = map[string]interface{}{"server": fix_addrs[idx]}
	return
}

// GetServer handle for server get for ktkt
func GetServerKt(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}
	params := r.URL.Query()
	key := params.Get("k")
	callback := params.Get("cb")
	protoStr := params.Get("p")
	res := map[string]interface{}{"ret": OK}
	defer retWrite(w, r, res, callback, time.Now())
	if key == "" {
		res["ret"] = ParamErr
		return
	}

	// 不再服务v1版本
	res["ret"] = ParamErr
	return

	// Match a push-server with the value computed through ketama algorithm
	node := myrpc.GetComet(key)
	if node == nil {
		res["ret"] = NotFoundServer
		return
	}
	addrs, ret := getProtoAddr(node, protoStr)
	if ret != OK {
		res["ret"] = ret
		return
	}

	// weisd 处理内外网ip对应
	map_list := map[string]string{}
	for _, m := range Conf.IpMaps {
		mArr := strings.Split(m, ":")
		map_list[mArr[0]] = mArr[1]
	}

	fix_addrs := make([]string, 0)
	for _, addr := range addrs {
		sArr := strings.Split(addr, ":")
		ip := sArr[0]
		port := sArr[1]
		if outAddr, ok := map_list[ip]; ok {
			ip = outAddr
		}

		// 添加数组
		fix_addrs = append(fix_addrs, ip+":"+port)
	}

	res["data"] = map[string]interface{}{"server": fix_addrs[0]}
	return
}
