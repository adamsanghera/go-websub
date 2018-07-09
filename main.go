package main

import (
	"fmt"
	"os"
	"time"

	"github.com/adamsanghera/go-websub/subscriber"
)

func main() {
	sub := subscriber.NewClient()

	fmt.Println(os.Args[1])

	sub.DiscoverTopic(os.Args[1])

	time.Sleep(1 * time.Second)

	fmt.Println(sub.GetHubsForTopic(os.Args[1]))

	sub.SubscribeToTopic(os.Args[1])

	time.Sleep(1 * time.Second)
}
