package bitmessage

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"time"
)

// This file implements functions for operating on list of nodes, including
// startup functions. Methods of Node can only be executed by the main server
// routine.

type streamNodes map[int]nodeMap

// nodeMap ...
type nodeMap map[ipPort]remoteNode

type remoteNode struct {
	// TODO: get rid of this conn.
	conn          net.Conn
	lastContacted time.Time
}

func (n *Node) addNode(stream int, ipPort ipPort, node remoteNode) {
	if _, ok := n.knownNodes[stream]; !ok {
		n.knownNodes[stream] = make(nodeMap)
	}
	n.knownNodes[stream][ipPort] = node
}

func (n *Node) delNode(stream int, ipPort ipPort) {
	if _, ok := n.knownNodes[stream]; !ok {
		n.knownNodes[stream] = make(nodeMap)
	}
	delete(n.knownNodes[stream], ipPort)
}

func (n *Node) bootstrap() {
	// Grab nodes from the config, add them to stream 1.
	n.knownNodes = make(streamNodes)
	for _, ipPort := range n.cfg.Nodes {
		go handshake(ipPort, remoteNode{}, n.resp)
	}

	// Add network bootstrap nodes to stream 1.
	for _, node := range findBootstrapNodes() {
		go handshake(node, remoteNode{}, n.resp)
	}
}

// findBootStrapNodes uses DNS resolution for finding bootstrap nodes for the
// network. The list of DNS hosts was obtained from the original client source
// in 2013-04-14. TODO: provide our own bootstrap nodes.
func findBootstrapNodes() (nodes []ipPort) {
	// XXX randomize.
	for _, node := range bootstrapNodes {
		if addrs, err := net.LookupIP(node[0]); err != nil {
			log.Printf("boot strap node lookup addr %v: error %v", node[0], err.Error())
		} else {
			if port, err := strconv.Atoi(node[streamOne]); err == nil {
				for _, addr := range addrs {
					nodes = append(nodes, ipPort(fmt.Sprintf("%v:%d", addr, port)))
				}
			}
		}
	}
	return nodes
}

func handshake(ipPort ipPort, node remoteNode, resp responses) {
	if !node.lastContacted.IsZero() && time.Since(node.lastContacted) < nodeConnectionRetryPeriod {
		// This node was contacted recently, so wait before the next try.
		return
	}

	node.lastContacted = time.Now()
	if node.conn == nil {
		var err error
		if node.conn, err = net.Dial("tcp", string(ipPort)); err != nil {
			log.Printf("error connecting to node %v: %v", ipPort, err)
			return
		}
		// TODO: move to a "ipPort to extendedNetworkAddress" function.
		tcpConn := node.conn.(*net.TCPConn)
		tcpAddr, err := net.ResolveTCPAddr("tcp", string(ipPort))
		var rawIp [16]byte
		copy(rawIp[:], tcpAddr.IP)
		resp.nodesChan <- []extendedNetworkAddress{{
			Time:   uint32(time.Now().Unix()),
			Stream: streamOne, // This should change after the version exchange.
			NetworkAddress: NetworkAddress{
				Services: ConnectionServiceNodeNetwork, //
				IP:       rawIp,
				Port:     uint16(tcpAddr.Port),
			}},
		}
		go handleConn(tcpConn, resp)
	}
	dest := node.conn.RemoteAddr().(*net.TCPAddr)
	go writeVersion(node.conn, dest)
}

// check logs the provided error if it's not nil.
func check(err error) {
	if err != nil {
		log.Fatalln("error", err.Error())
	}
}
