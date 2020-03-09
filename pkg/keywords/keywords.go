package keywords

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	c "stillgrove.com/gofeedyourself/pkg/collection"
	gwc "stillgrove.com/gofeedyourself/pkg/woocommerce/client"
)

func countWordCombinations(input [][]string) map[string]map[string]int {
	counts := make(map[string]map[string]int)

	previous, current := "NULL", "NULL"
	counts["NULL"] = make(map[string]int)
	for product := range input {
		for sentence := range input[product] {
			words := strings.Split(input[product][sentence], " ")
			for word := range words {
				if len(words[word]) <= 1 {
					continue
				}
				current = words[word]

				_, exist := counts[current]
				if exist == false {
					counts[current] = make(map[string]int)
				}
				counts[previous][current]++

				previous = current
			}
		}
	}

	return counts
}

func stripProducts(arr []gwc.Product) productRows {
	var vars = productRows{
		names:        make([]string, len(arr)),
		brands:       make([]string, len(arr)),
		descriptions: make([][]string, len(arr)),
		values:       make([]int32, len(arr)),
	}

	var found bool
	for i := range arr {
		for j := range arr[i].Brands {
			found = false
			for k, v := range arr[i].Brands[j].(map[string]interface{}) {
				if k == "name" {
					vars.brands[i] = strings.Trim(v.(string), " ")
					found = true
					break
				}

				if found == true {
					break
				}
			}
		}
		vars.names[i] = c.SanitizeHard(arr[i].Name)
		vars.descriptions[i] = strings.Split(c.SanitizeHard(arr[i].Description), ".")
		vars.values[i] = arr[i].MenuOrder
	}
	return vars
}

type productRows struct {
	names        []string
	brands       []string
	descriptions [][]string
	values       []int32
}

func handle(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func loadProducts(w *gwc.Client) (pr productRows, err error) {
	products, err := w.GetAllProducts("sv_se", false)
	if err != nil {
		return pr, err
	}

	pr = stripProducts(products)

	return pr, nil
}

func GenerateKeywordsFromWoo(w *gwc.Client) (keywords []map[string]string, err error) {
	/*var r = gwc.WooGetRequest{
	      Endpoint: "/wp-json/wc/v3/brands/",
	  }
	  raw, err := r.Send(&w)
	  handle(err)

	  var brands []interface{}
	  err = json.Unmarshal(raw, &brands)
	  handle(err)*/

	p, err := loadProducts(w)
	if err != nil {
		return nil, err
	}

	keywords = make([]map[string]string, len(p.names))

	for i := 0; i < len(p.names); i += 3 {
		keywords[i]["brand"] = p.brands[i]
		keywords[i]["name"] = p.names[i]
		keywords[i]["brand+name"] = fmt.Sprintf("%s %s", p.brands[i], p.names[i])
	}

	return keywords, nil
}
