package naming

import (
	"fmt"
	"math/rand"
)

var adjectives = []string{
	"admiring", "adoring", "agitated", "amazing", "angry", "awesome", "beautiful",
	"blissful", "bold", "boring", "brave", "busy", "charming", "clever", "cool",
	"compassionate", "competent", "condescending", "cranky", "crazy", "dazzling",
	"determined", "distracted", "dreamy", "eager", "ecstatic", "elastic", "elated",
	"elegant", "eloquent", "epic", "exciting", "fervent", "festive", "flamboyant",
	"focused", "friendly", "frosty", "funny", "gallant", "gifted", "goofy", "gracious",
	"grave", "happy", "hardcore", "heuristic", "hopeful", "hungry", "hysterical",
	"infinitesimal", "inspiring", "intelligent", "interesting", "jolly", "jovial",
	"keen", "kind", "laughing", "loving", "lucid", "luminous", "mad", "magical",
	"mystifying", "modest", "musing", "naughty", "nervous", "nice", "nifty", "nostalgic",
	"objective", "optimistic", "peaceful", "pedantic", "pensive", "practical", "priceless",
	"quirky", "quizzical", "reverent", "romantic", "sad", "serene", "sharp", "silly",
	"sleepy", "stoic", "strange", "stupefied", "suspicious", "sweet", "tender", "thrifty",
	"trusting", "unruffled", "upbeat", "vibrant", "vigilant", "vigorous", "wizardly",
	"wonderful", "zealous",
}

var names = []string{
	// Scientists & Physicists
	"einstein", "tesla", "newton", "curie", "fermi", "galileo", "kepler", "bohr",
	"hawking", "feynman", "planck", "oppenheimer", "schrodinger", "heisenberg",

	// Mathematicians & Computer Scientists
	"turing", "lovelace", "babbage", "knuth", "dijkstra", "hopper", "neumann",
	"ramanujan", "leibniz", "pascal", "riemann", "gauss", "euler", "chatelet",

	// Explorers & Astronomers
	"darwin", "columbus", "magellan", "hubble", "sagan", "copernicus", "brahe",
	"armstrong", "gagarin", "tereshkova", "aldrin",

	// Thinkers & Innovators
	"aristotle", "socrates", "plato", "da_vinci", "pasteur", "edison", "bell",
	"wright", "franklin", "mendeleev", "nobel", "carson", "goodall",
}

// GenerateDockerName returns a Docker-like friendly name
func GenerateVMName() string {
	// math/rand is automatically seeded in modern Go
	adj := adjectives[rand.Intn(len(adjectives))]
	name := names[rand.Intn(len(names))]

	return fmt.Sprintf("%s_%s", adj, name)
}
