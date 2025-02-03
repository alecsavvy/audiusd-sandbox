package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/AudiusProject/audiusd/pkg/core/server"
	client "github.com/cometbft/cometbft/rpc/client/http"
)

type PeerInfo struct {
	endpoint         string
	ip               string
	p2pOpen          bool
	nodeID           string
	currentPeerCount int
}

func (pi *PeerInfo) toConnStr() string {
	return fmt.Sprintf("%s@%s:26656", pi.nodeID, pi.ip)
}

func run() error {
	// Define your server's endpoint
	url := "https://creatornode2.audius.co/core/nodes/verbose"

	// Make the GET request
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Unmarshal JSON into the struct
	var response server.RegisteredNodesVerboseResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}

	// nodeid -> PeerInfo
	nodes := make(map[string]*PeerInfo)
	var nodesMU sync.Mutex

	for _, node := range response.RegisteredNodes {
		nodes[strings.ToLower(node.CometAddress)] = &PeerInfo{
			nodeID:   strings.ToLower(node.CometAddress),
			endpoint: node.Endpoint,
		}
	}

	var gatherPeerG sync.WaitGroup
	gatherPeerG.Add(len(response.RegisteredNodes))

	for _, node := range response.RegisteredNodes {
		go func(node *server.RegisteredNodeVerboseResponse) {
			defer func() {
				fmt.Printf("finished node: %s\n", node.Endpoint)
				gatherPeerG.Done()
			}()

			ctx := context.Background()
			rpc, err := client.New(fmt.Sprintf("%s/core/debug/comet", node.Endpoint))
			if err != nil {
				fmt.Printf("couldnt create comet rpc: %s %v\n", node.Endpoint, err)
				return
			}

			netInfo, err := rpc.NetInfo(ctx)
			if err != nil {
				fmt.Printf("couldn't get net info: %s %v\n", node.Endpoint, err)
				return
			}

			for _, peer := range netInfo.Peers {
				nodeID := peer.NodeInfo.DefaultNodeID
				ip := peer.RemoteIP
				nodesMU.Lock()
				node, exists := nodes[string(nodeID)]
				if exists {
					node.ip = ip
				}
				nodesMU.Unlock()
			}

		}(node)
	}

	gatherPeerG.Wait()

	var checkPortWG sync.WaitGroup
	checkPortWG.Add(len(nodes))

	for _, node := range nodes {
		go func(node *PeerInfo) {
			defer func() {
				checkPortWG.Done()
			}()

			if node.ip == "" {
				return
			}

			address := fmt.Sprintf("%s:%d", node.ip, 26656)
			conn, err := net.DialTimeout("tcp", address, 3*time.Second)
			if err != nil {
				fmt.Printf("port 26656 on node %s unreachable\n", node.endpoint)
				return
			}

			node.p2pOpen = true
			conn.Close()
		}(node)
	}

	checkPortWG.Wait()

	var nodeList []*PeerInfo
	for _, node := range nodes {
		nodeList = append(nodeList, node)
	}

	sort.Slice(nodeList, func(i, j int) bool {
		if nodeList[i].p2pOpen != nodeList[j].p2pOpen {
			return nodeList[i].p2pOpen
		}
		return nodeList[i].nodeID < nodeList[j].nodeID
	})

	writeToCSV("nodes.csv", nodeList)
	return nil
}

func writeToCSV(filename string, nodes []*PeerInfo) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write CSV header
	if err := writer.Write([]string{"NodeID", "Endpoint", "IP", "PortOpen"}); err != nil {
		return err
	}

	// Write node data
	for _, node := range nodes {
		portStatus := "false"
		if node.p2pOpen {
			portStatus = "true"
		}
		record := []string{node.nodeID, node.endpoint, node.ip, portStatus}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("crash: %v", err)
	}
}
