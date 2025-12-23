package app

import "math/rand"

// SecretWords is a curated list of words that work well for the game
// Themed around cyberpunk/tech but also includes common objects
var SecretWords = []string{
	// Cyberpunk / Tech
	"hacker", "cyborg", "android", "hologram", "matrix",
	"neon", "chrome", "synth", "glitch", "virus",
	"laser", "plasma", "quantum", "binary", "pixel",
	"drone", "robot", "avatar", "firewall", "bitcoin",
	"server", "arcade", "console", "joystick", "keyboard",
	"monitor", "circuit", "antenna", "satellite", "radar",

	// Animals
	"dragon", "phoenix", "unicorn", "kraken", "serpent",
	"tiger", "falcon", "wolf", "panther", "cobra",
	"dolphin", "octopus", "scorpion", "spider", "beetle",

	// Places
	"casino", "subway", "rooftop", "alley", "warehouse",
	"temple", "fortress", "pyramid", "bunker", "tower",
	"bridge", "tunnel", "harbor", "factory", "stadium",

	// Objects
	"diamond", "crystal", "mirror", "shadow", "blade",
	"helmet", "shield", "gauntlet", "compass", "lantern",
	"whistle", "umbrella", "hammer", "anchor", "hourglass",

	// Food & Drinks
	"coffee", "whiskey", "sushi", "burger", "pizza",
	"chocolate", "vanilla", "cinnamon", "wasabi", "honey",

	// Nature
	"thunder", "lightning", "tornado", "volcano", "glacier",
	"meteor", "eclipse", "aurora", "tsunami", "avalanche",

	// Abstract / Concepts
	"phantom", "specter", "enigma", "paradox", "illusion",
	"chaos", "harmony", "velocity", "gravity", "infinity",

	// Music / Art
	"rhythm", "melody", "symphony", "canvas", "sculpture",
	"graffiti", "tattoo", "mosaic", "origami", "kaleidoscope",
}

// GetRandomWord returns a random word from the secret words list
func GetRandomWord() string {
	return SecretWords[rand.Intn(len(SecretWords))]
}

// GetRandomWordExcluding returns a random word that's not in the excluded list
func GetRandomWordExcluding(excluded []string) string {
	excludeMap := make(map[string]bool)
	for _, w := range excluded {
		excludeMap[w] = true
	}

	// Try to find a non-excluded word
	for attempts := 0; attempts < 100; attempts++ {
		word := GetRandomWord()
		if !excludeMap[word] {
			return word
		}
	}

	// Fallback: just return any word
	return GetRandomWord()
}
