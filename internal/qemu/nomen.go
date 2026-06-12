package qemu

import (
	"fmt"
	"math/rand"
	"time"
)

// Lists of adjectives and famous scientists/engineers
var adjectives = []string{
	"admiring", "adoring", "agitated", "amazing", "angry",
	"awesome", "blissful", "boring", "brave", "clever",
}

var names = []string{
	"einstein", "tesla", "newton", "curie", "turing",
	"darwin", "galileo", "lovelace", "kepler", "fermi",
}

// GenerateDockerName returns a Docker-like friendly name
func GenerateVMName() string {
	rand.Seed(time.Now().UnixNano())
	adj := adjectives[rand.Intn(len(adjectives))]
	name := names[rand.Intn(len(names))]
	return fmt.Sprintf("%s_%s", adj, name)
}

// func NameGen() {
// 	for range 5 {
// 		fmt.Println(GenerateDockerName())
// 	}
// }
