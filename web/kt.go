package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "code.google.com/p/log4go"
	"github.com/Unknwon/com"
	// "github.com/hprose/hprose-go/hprose"
	myrpc "github.com/weisd/gopush-cluster/rpc"
	client "github.com/weisd/ktkt_client/go"
)

func InitRpcCient() {
	if len(Conf.RpcServer) == 0 {
		log.Error("RpcServer not found")
		panic("RpcServer not found")
	}
	client.InitClient(Conf.RpcServer)
	log.Info("连接rpc服务器 : %s", Conf.RpcServer)
	// c := hprose.NewClient(Conf.RpcServer)
	// c.UseService(&client.RbacClient)
}

func EncodeMd5(str string) string {
	m := md5.New()
	m.Write([]byte(str))
	return hex.EncodeToString(m.Sum(nil))
}

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

	idx := randSource.Intn(len(fix_addrs))

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

// 接收 kt服务端结果
func PushKtResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}
	body := ""
	res := map[string]interface{}{"ret": OK}
	defer retPWrite(w, r, res, &body, time.Now())
	// post param
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		res["ret"] = InternalErr
		log.Error("ioutil.ReadAll() failed (%v)", err)
		return
	}
	body = string(bodyBytes)
	msgBytes, strategyId, ret := parseMultiPrivate(bodyBytes)
	if ret != OK {
		res["ret"] = ret
		return
	}
	rm := json.RawMessage(msgBytes)
	msg, err := rm.MarshalJSON()
	if err != nil {
		res["ret"] = ParamErr
		log.Error("json.RawMessage(\"%s\").MarshalJSON() error(%v)", string(msg), err)
		return
	}

	if len(strategyId) == 0 {
		res["ret"] = ParamErr
		return
	}
	// weisd
	uids, err := client.RbacClient.GetUserIdsByNodeId(strategyId[0], client.PERMISSION_TYPE_STRATEGY)
	if err != nil {
		log.Error("%s, GetUserIdsByNodeId failed : %v", err)
		return
	}

	prefix := Conf.CHANNEL_PREFIX

	keys := make([]string, 0, len(uids))

	// 只发送在发送列表中的，取交集
	sendUids := client.RbacClient.GetStockSendUidsByStrategyId(strategyId[0])

	// 只发送在sendUids中的用户
	for _, uid := range uids {
		for i, l := len(sendUids); i < l; i++ {
			if uid == sendUids[i] {
				uidStr := com.ToStr(uid)

				keys = append(keys, EncodeMd5(prefix+uidStr))
			}
		}
	}

	// url param
	params := r.URL.Query()
	expire, err := strconv.ParseUint(params.Get("expire"), 10, 32)
	if err != nil {
		res["ret"] = ParamErr
		log.Error("strconv.ParseUint(\"%s\", 10, 32) error(%v)", params.Get("expire"), err)
		return
	}
	// match nodes
	nodes := map[*myrpc.CometNodeInfo]*[]string{}
	for i := 0; i < len(keys); i++ {
		node := myrpc.GetComet(keys[i])
		if node == nil || node.Rpc == nil {
			res["ret"] = NotFoundServer
			return
		}
		keysTmp, ok := nodes[node]
		if ok {
			*keysTmp = append(*keysTmp, keys[i])
		} else {
			nodes[node] = &([]string{keys[i]})
		}
	}
	var fKeys []string
	//push to every node
	for cometInfo, ks := range nodes {
		client := cometInfo.Rpc.Get()
		if client == nil {
			log.Error("cannot get comet rpc client")
			fKeys = append(fKeys, *ks...)
			continue
		}
		args := &myrpc.CometPushPrivatesArgs{Msg: json.RawMessage(msg), Expire: uint(expire), Keys: *ks}
		resp := myrpc.CometPushPrivatesResp{}
		if err := client.Call(myrpc.CometServicePushPrivates, args, &resp); err != nil {
			log.Error("client.Call(\"%s\", \"%v\", &ret) error(%v)", myrpc.CometServicePushPrivates, args.Keys, err)
			fKeys = append(fKeys, *ks...)
			continue
		}
		log.Debug("fkeys len(%d) addr:%v", len(resp.FKeys), cometInfo.RpcAddr)
		fKeys = append(fKeys, resp.FKeys...)
	}
	res["ret"] = OK
	if len(fKeys) != 0 {
		res["data"] = map[string]interface{}{"fk": fKeys}
	}
	return
}
