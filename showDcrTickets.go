// Example of usage of the dcrwallet API to gather data on the tickets (for the locally open wallet)
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"fmt"
	"github.com/decred/dcrd/chaincfg"
	pb "github.com/decred/dcrwallet/rpc/walletrpc"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io"
	"os/user"
	"time"
)

// Location of your certificate
var certificateFile = ".config/decrediton/rpc.cert"

func reverse(numbers []byte) []byte {
	for i, j := 0, len(numbers)-1; i < j; i, j = i+1, j-1 {
		numbers[i], numbers[j] = numbers[j], numbers[i]
	}
	return numbers
}

type ticketState int

const (
	immature ticketState = iota
	inPool
	voted
	immature2
	expired
	unmined // for later
)

type ticketData struct {
	timestamp     int64
	height        int32
	state         ticketState
	voteTimestamp int64
	voteHeight    int32
}

func main() {
	usr, err := user.Current()
	if err != nil {
		fmt.Println(err)
	}
	fullCertificateFile := usr.HomeDir + "/" + certificateFile

	creds, err := credentials.NewClientTLSFromFile(fullCertificateFile, "localhost")
	if err != nil {
		fmt.Println(err)
		return
	}
	conn, err := grpc.Dial("localhost:9121", grpc.WithTransportCredentials(creds))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()
	c := pb.NewWalletServiceClient(conn)

	// Lask block
	accountsRequest := &pb.AccountsRequest{}
	accountsResponse, err := c.Accounts(context.Background(), accountsRequest)
	if err != nil {
		fmt.Println(err)
		return
	}
	currentBlockHeight := accountsResponse.CurrentBlockHeight

    // Map of all found tickets
	tickets := make(map[string]*ticketData)
    // Ordered list of ticket purchase transations hashes to keep the order 
	var ticketsKeys []string

    // Initial block height of the search (higher means slightly lower search time if you know your older ticket heigth)
	var initialHeight int32 = 1
	cntTicket := 0
	cntVote := 0
	cntExpired := 0

	for {
		getTransactionsRequest := &pb.GetTransactionsRequest{
			StartingBlockHeight: initialHeight,
		}
        // Get all transanction related to the opened wallet
		getTransactionsResponse, err := c.GetTransactions(context.Background(), getTransactionsRequest)
		if err != nil {
			fmt.Println(err)
			return
		}
		for {
			b, err := getTransactionsResponse.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Println(err)
				break
			}
			for _, tr := range b.MinedTransactions.Transactions {
				if tr.TransactionType == 1 {
					// That a ticket purchase
					cntTicket++
					transactionHash := hex.EncodeToString(reverse(tr.Hash))
					ticketsKeys = append(ticketsKeys, transactionHash)
					tickets[transactionHash] = new(ticketData)
					tickets[transactionHash] = &ticketData{
						timestamp: b.MinedTransactions.Timestamp,
						height:    b.MinedTransactions.Height}
					if currentBlockHeight-b.MinedTransactions.Height < int32(chaincfg.MainNetParams.TicketMaturity) {
						// TODO < or <= ? (for all)
						tickets[transactionHash].state = immature
					} else if currentBlockHeight-b.MinedTransactions.Height > int32(chaincfg.MainNetParams.TicketExpiry) {
						cntExpired++
						tickets[transactionHash].state = expired
					} else {
						// State that will be overwritten if a corresponding Vote is found
						tickets[transactionHash].state = inPool
					}
				}
				if tr.TransactionType == 2 {
					// That's a vote
					cntVote++
					voteTransationBytes := tr.Transaction[46:(46 + 32)]
					voteTransaction := hex.EncodeToString(reverse(voteTransationBytes))
					_, ok := tickets[voteTransaction]
					if ok {
						tickets[voteTransaction].voteTimestamp = tr.Timestamp
						tickets[voteTransaction].voteHeight = b.MinedTransactions.Height

						if currentBlockHeight-b.MinedTransactions.Height < 256 {
							tickets[voteTransaction].state = immature2
						} else {
							tickets[voteTransaction].state = voted
						}
					} else {
						fmt.Println("Err : Incoherent ticket found in vote transaction.")
					}
				}
			}
		}
		break
	}
    // Display of data
	fmt.Println("You have ", cntTicket, "tickets. ", cntVote, " of them have voted.\n")

	i := 1
	totalDaysVoted := 0.0
	totalDaysNotVotedYet := 0.0
	for _, k := range ticketsKeys {
		t := tickets[k]
		fmt.Println("Ticket #", i)
		var status string
		switch t.state {
		case immature:
			status = "Immature after inclusion in a block"
		case inPool:
			status = "Live"
		case immature2:
			status = "Immature after vote"
		case voted:
			status = "Voted and paid"
		case expired:
			status = "Expired"
		}
		fmt.Println("\tStatus: ", status)
		fmt.Println("\tTicket Height : ", t.height)
		if t.voteHeight != 0 {
			// Voted tickets
			fmt.Println("\tVote Height :   ", t.voteHeight)
			duration := time.Unix(t.voteTimestamp, 0).Sub(time.Unix(t.timestamp, 0))
			days := duration.Hours() / 24
			totalDaysVoted += days
			fmt.Printf("\tAge when voted : %.1f days\n", days)
		} else {
			// Tickets not voted yet
			if t.state != expired {
				duration := time.Since(time.Unix(t.timestamp, 0))
				days := duration.Hours() / 24
				totalDaysNotVotedYet += days
				fmt.Printf("\tAge  :           %.1f days\n", days)
			}
		}
		i++
	}
	fmt.Printf("\nMean time in the pool for voted tickets is %.1f days.", totalDaysVoted/float64(cntVote))
	fmt.Printf("\nMean time in the pool for live and immature tickets is %.1f days.", totalDaysNotVotedYet/float64(cntTicket-cntVote-cntExpired))
}
