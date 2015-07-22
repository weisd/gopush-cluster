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
	"bufio"
	log "code.google.com/p/log4go"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	// "math/rand"
	"net"
	"net/http"
	"strconv"
	"time"
)

func main() {

	var err error
	flag.Parse()
	// init config
	Conf, err = InitConfig(ConfFile)
	if err != nil {
		log.Error("NewConfig(\"%s\") failed (%s)", ConfFile, err.Error())
		return
	}

	log.LoadConfiguration(Conf.Log)
	defer log.Close()

	wait := make(chan bool)

	var i int64
	for i = 0; i < Conf.Loop; i++ {
		go Client(i)
		time.Sleep(50 * time.Millisecond)
	}

	<-wait
}
func Client(idx int64) {
	first := false

	// 重新请求连接
restart:

	// restartSec := rand.Intn(20)

	if !first {
		time.Sleep(10 * time.Second)
	}

	key := fmt.Sprintf("%d", idx)

	server := ServerGet(Conf.Api, key, idx)
	if len(server) == 0 {
		goto restart
	}

	addr, err := net.ResolveTCPAddr("tcp", server)
	if err != nil {
		log.Error("net.ResolveTCPAddr(\"tcp\", \"%s\") failed (%s)  [%d]", Conf.Addr, err.Error(), idx)
		goto restart

	}
	log.Info("connect to gopush-cluster comet [%d]", idx)
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		log.Error("net.DialTCP() failed (%s)", err.Error())
		goto restart

	}
	log.Info("send sub request [%d]", idx)
	proto := []byte(fmt.Sprintf("*3\r\n$3\r\nsub\r\n$%d\r\n%s\r\n$%d\r\n%d\r\n", len(Conf.Key), Conf.Key, len(strconv.Itoa(int(Conf.Heartbeat))), Conf.Heartbeat))
	log.Info("send protocol: %s [%d]", string(proto), idx)
	if _, err := conn.Write(proto); err != nil {
		log.Error("conn.Write() failed (%s) [%d]", err.Error(), idx)
		conn.Close()
		goto restart

	}
	// get first heartbeat

	rd := bufio.NewReader(conn)
	// block read reply from service
	log.Info("wait message [%d]", idx)
	for {
		if err := conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(Conf.Heartbeat) * 2)); err != nil {
			log.Error("conn.SetReadDeadline() failed (%s) [%d]", err.Error(), idx)
			conn.Close()
			goto restart
		}
		line, err := rd.ReadBytes('\n')
		if err != nil {
			log.Error("rd.ReadBytes() failed (%s) [%d]", err.Error(), idx)
			conn.Close()
			goto restart
		}
		if line[len(line)-2] != '\r' {
			log.Error("protocol reply format error [%d]", idx)
			conn.Close()
			goto restart
		}
		log.Info("line: %s", line)
		switch line[0] {
		// reply
		case '$':
			cmdSize, err := strconv.Atoi(string(line[1 : len(line)-2]))
			if err != nil {
				log.Error("protocol reply format error [%d]", idx)
				conn.Close()
				goto restart
			}
			data, err := rd.ReadBytes('\n')
			if err != nil {
				log.Error("protocol reply format error [%d]", idx)
				conn.Close()
				goto restart
			}
			if len(data) != cmdSize+2 {
				log.Error("protocol reply format error: %s [%d]", data, idx)
				conn.Close()
				goto restart
			}
			if data[cmdSize] != '\r' || data[cmdSize+1] != '\n' {
				log.Error("protocol reply format error [%d]", idx)
				conn.Close()
				goto restart
			}
			reply := string(data[0:cmdSize])
			log.Info("receive msg: %s [%d]", reply, idx)
			break
			// heartbeat
		case '+':
			if !first {
				// send heartbeat
				go func() {
					for {
						log.Info("send heartbeat")
						if _, err := conn.Write([]byte("h")); err != nil {
							log.Error("conn.Write() failed (%s) [%d]", err.Error(), idx)
							conn.Close()
							return
						}
						time.Sleep(time.Duration(Conf.Heartbeat) * time.Second)
					}
				}()
				first = true
			}
			log.Info("receive heartbeat [%d]", idx)
			break
		}
	}
}

func ServerGet(api, key string, idx int64) (server string) {
	var err error
	// 取
	serverApi := fmt.Sprintf("http://%s/kt2/server/get?k=%s&p=2", api, key)
	fmt.Println("开始请求server get [%d]", idx)
	resp, err := http.Get(serverApi)
	if err != nil {
		fmt.Println("请求server/get失败[%d]", err.Error(), idx)
		return
	}

	defer resp.Body.Close()

	resBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取body失败 [%d]", idx)
		return
	}

	var resonse map[string]interface{}

	err = json.Unmarshal(resBody, &resonse)
	if err != nil {
		fmt.Println(`解析 body失败 [%d]`, idx)
		return
	}

	fmt.Println("resonse", resonse)

	ret, ok := resonse["ret"].(float64)
	if !ok {
		fmt.Println("ret 格式不正确 [%d]", resonse["ret"], idx)
		return
	}

	if ret > 0 {
		fmt.Println("取comet地址失败 [%d]", ret, idx)
		return
	}

	data := resonse["data"].(map[string]interface{})
	// if !ok {
	// 	fmt.Println("data格式不正确 ", resonse["data"])
	// 	return
	// }

	server = data["server"].(string)
	return
}
