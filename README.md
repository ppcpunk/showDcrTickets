# showDcrTickets

This is a very simple program to gather some data about the purchased tickets of your own Decred wallet (if open).

Note that this is my first time using any of Go, gRPC and dcrwallet API so my usage of them can be sub-optimal. Purpose of this program is mostly learning them.

### Usage 
Your Decred wallet have to be in use.
The call to grpc.Dial may have to be adapted to your configuration. Path to certificate also have to be changed if you are not using a Unix system or changed the default path. Configured by default for Decredition.

### TODO
- Consider not yet mined tickets (and those not mined and refunded ?)
- Test edge cases
- Improve output formating, create an HTML page


### Example of output
```
You have 10 tickets. 6 of them have voted.

Ticket # 1
	Status:  Voted and paid
	Ticket Height :  132368
	Vote Height :    136325
	Age when voted : 15.5 days
Ticket # 2
	Status:  Live
	Ticket Height :  137349
	Age  :           20.4 days
[...]

Mean time in the pool for voted tickets is 18.9 days.
Mean time in the pool for live and immature tickets is 26.8 days.
```

If anyone wants to tip : DsTWXfhDqVjKRdyJfysLuvcTzm9UMjHDbfY
