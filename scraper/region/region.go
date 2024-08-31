package region

import (
	"fmt"
	"iter"
)

// Constants representing region IDs.
const (
	Invalid        = ID(0)
	Crimea         = ID(1)
	Vinnytsia      = ID(2)
	Volyn          = ID(3)
	Dnipro         = ID(4)
	Donetsk        = ID(5)
	Zhytomyr       = ID(6)
	Zakarpattia    = ID(7)
	Zaporizhzhia   = ID(8)
	IvanoFrankivsk = ID(9)
	Kyiv           = ID(10)
	Kirovohrad     = ID(11)
	Luhansk        = ID(12)
	Lviv           = ID(13)
	Mykolaiv       = ID(14)
	Odesa          = ID(15)
	Poltava        = ID(16)
	Rivne          = ID(17)
	Sumy           = ID(18)
	Ternopil       = ID(19)
	Kharkiv        = ID(20)
	Kherson        = ID(21)
	Khmelnytskyi   = ID(22)
	Cherkasy       = ID(23)
	Chernivtsi     = ID(24)
	Chernihiv      = ID(25)
	KyivCity       = ID(26)
	SevastopolCity = ID(27)
)

var namesById = map[ID]string{
	1:  "Автономна Республіка Крим",
	2:  "Вінницька область",
	3:  "Волинська область",
	4:  "Дніпропетровська область",
	5:  "Донецька область",
	6:  "Житомирська область",
	7:  "Закарпатська область",
	8:  "Запорізька область",
	9:  "Івано-Франківська область",
	10: "Київська область",
	11: "Кіровоградська область",
	12: "Луганська область",
	13: "Львівська область",
	14: "Миколаївська область",
	15: "Одеська область",
	16: "Полтавська область",
	17: "Рівненська область",
	18: "Сумська область",
	19: "Тернопільська область",
	20: "Харківська область",
	21: "Херсонська область",
	22: "Хмельницька область",
	23: "Черкаська область",
	24: "Чернівецька область",
	25: "Чернігівська область",
	26: "м. Київ",
	27: "м. Севастополь",
}

var idsByName = make(map[string]ID, len(namesById))

func init() {
	for id, name := range namesById {
		idsByName[name] = id
	}
}

// Parse converts a region name to its corresponding ID.
// Returns an error if the name is not found.
func Parse(name string) (ID, error) {
	if id, exists := idsByName[name]; exists {
		return id, nil
	}
	return Invalid, fmt.Errorf("name '%s' doesn't exist", name)
}

// Count returns the number of regions in the package.
func Count() int {
	return len(namesById)
}

// Iterator returns an iterator over region IDs and names.
func Iterator() iter.Seq2[ID, string] {
	return func(yield func(ID, string) bool) {
		for id, name := range namesById {
			if !yield(id, name) {
				return
			}
		}
	}
}

// ID represents a unique identifier for a region.
type ID int

// String returns the name of the region corresponding to the ID.
// Returns an empty string if the ID is invalid.
func (id ID) String() string {
	if name, exists := namesById[id]; exists {
		return name
	}
	return ""
}
