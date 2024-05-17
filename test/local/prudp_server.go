package main

import (
	"fmt"

	"github.com/PretendoNetwork/nex-go/v2"
)

func main() {

	fmt.Println("Starting server")

	server := nex.NewPRUDPServer()
	server.LibraryVersions.SetDefault(nex.NewLibraryVersion(1, 1, 0))
	server.SetFragmentSize(962)
	server.SessionKeyLength = 16
	server.AccessKey = "foo"

	endpoint := nex.NewPRUDPEndPoint(1)
	server.BindPRUDPEndPoint(endpoint)
	endpoint.OnData(func(packet nex.PacketInterface) {
	})

	server.Listen(12345)
}
