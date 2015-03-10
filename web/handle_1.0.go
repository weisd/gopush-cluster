// Copyright © 2014 Terry Mao, LiuDing All rights reserved.
// This file is part of gopush-cluster.

// gopush-cluster is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// gopush-cluster is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with gopush-cluster.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	log "code.google.com/p/log4go"
	myrpc "github.com/weisd/gopush-cluster/rpc"
	"net/http"
	"strconv"
	"strings"
	"time"
)

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
	log.Error("ipmaps : %v", Conf.IpMaps)
	log.Error("addrs : %v", addrs)

	map_list := map[string]string{}
	for _, m := range Conf.IpMaps {
		mArr := strings.Split(m, ":")
		map_list[mArr[0]] = mArr[1];
	}

	log.Error("map_list : %v", map_list)

	fix_addrs := make([]string, len(addrs))
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

// GetServer handle for server get
func GetServer(w http.ResponseWriter, r *http.Request) {
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
	res["data"] = map[string]interface{}{"server": addrs[0]}
	return
}

// GetServer2 handle for server get.
func GetServer2(w http.ResponseWriter, r *http.Request) {
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
	// give client a addr list, client do the best choice
	res["data"] = map[string]interface{}{"server": addrs}
	return
}

// GetOfflineMsg get offline mesage http handler.
func GetOfflineMsg(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}
	params := r.URL.Query()
	key := params.Get("k")
	midStr := params.Get("m")
	callback := params.Get("cb")
	res := map[string]interface{}{"ret": OK}
	defer retWrite(w, r, res, callback, time.Now())
	if key == "" || midStr == "" {
		res["ret"] = ParamErr
		return
	}
	mid, err := strconv.ParseInt(midStr, 10, 64)
	if err != nil {
		res["ret"] = ParamErr
		log.Error("strconv.ParseInt(\"%s\", 10, 64) error(%v)", midStr, err)
		return
	}
	// RPC get offline messages
	reply := &myrpc.MessageGetResp{}
	args := &myrpc.MessageGetPrivateArgs{MsgId: mid, Key: key}
	client := myrpc.MessageRPC.Get()
	if client == nil {
		log.Error("no message node found")
		res["ret"] = InternalErr
		return
	}
	if err := client.Call(myrpc.MessageServiceGetPrivate, args, reply); err != nil {
		log.Error("myrpc.MessageRPC.Call(\"%s\", \"%v\", reply) error(%v)", myrpc.MessageServiceGetPrivate, args, err)
		res["ret"] = InternalErr
		return
	}
	if len(reply.Msgs) == 0 {
		return
	}
	res["data"] = map[string]interface{}{"msgs": reply.Msgs}
	return
}

// GetTime get server time http handler.
func GetTime(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}
	params := r.URL.Query()
	callback := params.Get("cb")
	res := map[string]interface{}{"ret": OK}
	now := time.Now()
	defer retWrite(w, r, res, callback, now)
	res["data"] = map[string]interface{}{"timeid": now.UnixNano() / 100}
	return
}
