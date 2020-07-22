package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/DCsunset/openwhisk-grpc/db"
	"google.golang.org/grpc"
)

func mapper(client db.DbServiceClient, sessionId int64, virtualLoc int64) {
	count := make(map[string]string)
	var keys []string
	key := strconv.Itoa(int(virtualLoc))
	keys = append(keys, key)

	res, _ := client.Get(context.Background(), &db.GetRequest{
		Keys: keys,
		Loc:  0,
	})

	words := strings.Fields(res.GetData()[key])
	for _, word := range words {
		c, ok := count[word]
		if ok {
			orig, _ := strconv.Atoi(c)
			count[word] = strconv.Itoa(orig + 1)
		} else {
			count[word] = "1"
		}
	}

	client.Set(context.Background(), &db.SetRequest{
		SessionId:  sessionId,
		Data:       count,
		VirtualLoc: virtualLoc,
		Dep:        -1,
	})

	fmt.Println("{ \"ok\": true }")
}

func makeRange(min, max int64) []int64 {
	a := make([]int64, max-min)
	for i := range a {
		a[i] = min + int64(i)
	}
	return a
}

func reducer(client db.DbServiceClient, sessionId int64) {
	count := make(map[string]string)

	// From key 0 to 20
	virtualLocs := makeRange(0, 20)

	for _, loc := range virtualLocs {
		res, _ := client.Get(context.Background(), &db.GetRequest{
			SessionId:  sessionId,
			Keys:       nil,
			Loc:        -2,
			VirtualLoc: loc,
		})
		partialData := res.GetData()
		for key, valueStr := range partialData {
			c, ok := count[key]
			if ok {
				orig, _ := strconv.Atoi(c)
				value, _ := strconv.Atoi(valueStr)
				count[key] = strconv.Itoa(orig + value)
			} else {
				count[key] = valueStr
			}
		}
	}

	res, _ := json.Marshal(count)
	fmt.Println(string(res))
}

const APIHOST = "172.18.0.4:31001"
const ACTION = "test-grpc"
const username = "23bc46b1-71f6-4ed5-8c54-816aa4f8c502"
const password = "123zO3xZCLrMN6v2BKK1dXYFpXlPkccOFqm12CdAsMgRU4VrNZ9lyGVCGuMDGIwP"

type Argument struct {
	// keys from [low, high)
	Kind       string `json:"kind"`
	SessionId  int64  `json:"sessionId,omitempty"`
	VirtualLoc int64  `json:"virtualLoc,omitempty"`
}

func callAction(params *Argument, blocking bool) {
	jsonValue, _ := json.Marshal(params)

	var url string
	if blocking {
		url = fmt.Sprintf("https://%s/api/v1/namespaces/guest/actions/%s?blocking=true&result=true", APIHOST, ACTION)
	} else {
		url = fmt.Sprintf("https://%s/api/v1/namespaces/guest/actions/%s", APIHOST, ACTION)
	}

	// No need to wait for the response
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	req.SetBasicAuth(username, password)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)

	if err != nil {
		log.Fatalf("Fail to invoke action: %v", err)
	}
	defer resp.Body.Close()

	if blocking {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		bodyString := string(bodyBytes)
		fmt.Println(bodyString)
	}
}

func runner(client db.DbServiceClient, sessionId int64) {
	// Count key from 0 to 20
	virtualLocs := makeRange(0, 20)

	// No need to wait for result
	for _, loc := range virtualLocs {
		callAction(&Argument{
			Kind:       "mapper",
			SessionId:  sessionId,
			VirtualLoc: loc,
		}, false)
	}

	// Reduce using virtual locations
	callAction(&Argument{
		Kind:      "reducer",
		SessionId: sessionId,
	}, true)
}

const address = "172.18.0.1:9000"

func main() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// parse json args
	var args Argument
	json.Unmarshal([]byte(os.Args[1]), &args)

	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Cannot connect: %v", err)
	}
	defer conn.Close()
	client := db.NewDbServiceClient(conn)

	if args.Kind == "runner" {
		// Generate new session ID
		id := rand.Int63()
		runner(client, id)
	} else if args.Kind == "mapper" {
		mapper(client, args.SessionId, args.VirtualLoc)
	} else {
		reducer(client, args.SessionId)
	}
}