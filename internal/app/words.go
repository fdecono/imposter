package app

import "math/rand"

// Language represents the game language
type Language string

const (
	LangEnglish Language = "en"
	LangSpanish Language = "es"
)

// EnglishWords - curated list of English words for the game
var EnglishWords = []string{
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

	// Games & Entertainment
	"chess", "poker", "dice", "cards", "domino", "roulette",

	// Weapons
	"katana", "revolver", "grenade", "cannon", "rifle", "crossbow", "dagger",

	// Vehicles
	"motorcycle", "helicopter", "submarine", "spaceship", "hovercraft", "rocket",

	// Gems & Materials
	"silver", "copper", "titanium", "platinum", "emerald", "sapphire",

	// Professions & Characters
	"ninja", "samurai", "pirate", "wizard", "knight", "assassin",

	// Time of Day
	"midnight", "sunrise", "sunset", "twilight", "dawn",

	// Mystical
	"oracle", "relic", "cipher", "amulet",
}

// SpanishWords - curated list of Spanish words for the game
var SpanishWords = []string{
	// Tecnología / Cyberpunk
	"hacker", "androide", "holograma", "matriz", "neón",
	"cromo", "virus", "láser", "plasma", "píxel",
	"dron", "robot", "servidor", "teclado", "pantalla",
	"circuito", "antena", "satélite", "radar", "código",

	// Animales
	"dragón", "fénix", "unicornio", "serpiente", "tigre",
	"halcón", "lobo", "pantera", "cobra", "delfín",
	"pulpo", "escorpión", "araña", "escarabajo", "águila",
	"tiburón", "cuervo", "murciélago", "camaleón", "jaguar",

	// Lugares
	"casino", "metro", "azotea", "callejón", "bodega",
	"templo", "fortaleza", "pirámide", "búnker", "torre",
	"puente", "túnel", "puerto", "fábrica", "estadio",
	"catedral", "laberinto", "caverna", "volcán", "castillo",

	// Objetos
	"diamante", "cristal", "espejo", "sombra", "espada",
	"casco", "escudo", "brújula", "linterna", "silbato",
	"paraguas", "martillo", "ancla", "reloj", "cadena",
	"corona", "llave", "máscara", "vela", "cofre",

	// Comida y Bebidas
	"café", "whisky", "tequila", "cerveza", "pizza",
	"chocolate", "vainilla", "canela", "miel", "limón",
	"mango", "sandía", "churro", "empanada", "tacos",

	// Naturaleza
	"trueno", "relámpago", "tornado", "volcán", "glaciar",
	"meteoro", "eclipse", "aurora", "tsunami", "avalancha",
	"tormenta", "huracán", "terremoto", "maremoto", "niebla",

	// Conceptos / Abstracto
	"fantasma", "espectro", "enigma", "paradoja", "ilusión",
	"caos", "armonía", "velocidad", "gravedad", "infinito",
	"destino", "misterio", "secreto", "leyenda", "profecía",

	// Música / Arte
	"ritmo", "melodía", "sinfonía", "lienzo", "escultura",
	"grafiti", "tatuaje", "mosaico", "origami", "mural",

	// Juegos y Entretenimiento
	"ajedrez", "póker", "dados", "cartas", "dominó", "ruleta",

	// Armas
	"katana", "revólver", "granada", "cañón", "rifle", "ballesta", "daga",

	// Vehículos
	"motocicleta", "helicóptero", "submarino", "nave", "cohete", "aeronave",

	// Gemas y Materiales
	"plata", "cobre", "titanio", "platino", "esmeralda", "zafiro",

	// Profesiones y Personajes
	"ninja", "samurái", "pirata", "mago", "caballero", "asesino",

	// Momentos del Día
	"medianoche", "amanecer", "atardecer", "crepúsculo", "alba",

	// Místico
	"oráculo", "reliquia", "cifra", "amuleto",
}

// GetRandomWord returns a random word in the specified language
func GetRandomWord(lang Language) string {
	words := getWordList(lang)
	return words[rand.Intn(len(words))]
}

// GetRandomWordExcluding returns a random word that's not in the excluded list
func GetRandomWordExcluding(lang Language, excluded []string) string {
	words := getWordList(lang)
	excludeMap := make(map[string]bool)
	for _, w := range excluded {
		excludeMap[w] = true
	}

	// Try to find a non-excluded word
	for attempts := 0; attempts < 100; attempts++ {
		word := words[rand.Intn(len(words))]
		if !excludeMap[word] {
			return word
		}
	}

	// Fallback: just return any word
	return words[rand.Intn(len(words))]
}

// getWordList returns the word list for a given language
func getWordList(lang Language) []string {
	switch lang {
	case LangSpanish:
		return SpanishWords
	default:
		return EnglishWords
	}
}

// ValidateLanguage checks if a language code is valid
func ValidateLanguage(lang string) Language {
	switch lang {
	case "es", "ES", "spanish", "Spanish":
		return LangSpanish
	default:
		return LangEnglish
	}
}
