package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"github.ibm.com/ei-agent/pkg/setupFrame"
)

var (
	listener          = flag.String("listen", ":5001", "listen host:port for client")
	servicenode       = flag.String("sn", "", "listen host:port of server side service node")
	destPort          = flag.String("destPort", "5003", "Destination IP")
	destIp            = flag.String("destIp", "127.0.0.1", "Destination port")
	maxDataBufferSize = 64 * 1024
)

func main() {
	flag.Parse()
	fmt.Println("********** Start Client ************")
	fmt.Printf("Strart client listen: %v  send to server: %v \n", *listener, *servicenode)
	if *listener == "" || *servicenode == "" {
		fmt.Println("missing listener or service")
		os.Exit(-1)
	}

	acceptLoop(*listener, *servicenode) // missing channel for signal handler
}

func acceptLoop(listener, servicenode string) error {
	// open listener
	acceptor, err := net.Listen("tcp", listener)
	if err != nil {
		return err
	}
	// loop until signalled to stop
	for {
		c, err := acceptor.Accept()
		if err != nil {
			return err
		}
		go dispatch(c, servicenode)
	}
}

func dispatch(c net.Conn, servicenode string) error {
	nodeConn, err := net.Dial("tcp", servicenode)
	if err != nil {
		return err
	}
	return ioLoop(c, nodeConn)
}

func ioLoop(cl, sn net.Conn) error {
	defer cl.Close()
	defer sn.Close()

	fmt.Println("Cient", cl.RemoteAddr().String(), "->", cl.LocalAddr().String())
	fmt.Println("Server", sn.LocalAddr().String(), "->", sn.RemoteAddr().String())
	done := &sync.WaitGroup{}
	done.Add(2)

	go clientToServer(done, cl, sn)
	go serverToClient(done, cl, sn)

	done.Wait()

	return nil
}

func clientToServer(wg *sync.WaitGroup, cl, sn net.Conn) error {

	defer wg.Done()
	var err error

	setupFrame.SendFrame(cl, sn, *destIp, *destPort) //Need to check performance impact
	fmt.Printf("[clientToServer]: Finish send SetupFrame \n")
	bufData := make([]byte, maxDataBufferSize)

	for {
		numBytes, err := cl.Read(bufData)
		if err != nil {
			if err == io.EOF {
				err = nil //Ignore EOF error
			} else {
				fmt.Printf("[clientToServer]: Read error %v\n", err)
			}

			break
		}

		_, err = sn.Write(bufData[:numBytes])
		if err != nil {
			fmt.Printf("[clientToServer]: Write error %v\n", err)
			break
		}
	}
	if err == io.EOF {
		return nil
	} else {
		return err
	}

}

func serverToClient(wg *sync.WaitGroup, cl, sn net.Conn) error {
	defer wg.Done()

	bufData := make([]byte, maxDataBufferSize)
	var err error
	for {
		numBytes, err := sn.Read(bufData)
		if err != nil {
			if err == io.EOF {
				err = nil //Ignore EOF error
			} else {
				fmt.Printf("[serverToClient]: Read error %v\n", err)
			}
			break
		}
		_, err = cl.Write(bufData[:numBytes])
		if err != nil {
			fmt.Printf("[serverToClient]: Write error %v\n", err)
			break
		}
	}
	return err
}

// allocate 4B frame-buffer and 64KB payload buffer
// forever {
//    read 4B into frame-buffer
//    if frame.Type == control { // not expected yet, except for error returns from SN
// 	     read and process control frame
//    } else {
// 	 	 read(sn, payload, frame.Len) // might require multiple reads and need a timeout deadline set
//	     send(cl, payload)
//    }
// }
