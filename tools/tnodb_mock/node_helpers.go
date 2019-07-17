package main

func nodeExist(nodeID string) bool {
	_, ok := nodeStore[nodeID]
	return ok
}
