package wooclient

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

type errorResponse struct {
	ID    int                    `json:"id"`
	Error map[string]interface{} `json:"error"`
}

func checkResult(input []byte) error {
	var (
		exist bool
	)
	if !strings.Contains(string(input), "error") {
		/*if strings.Contains(string(input), "status") {
			return fmt.Errorf("%s", string(input))
		}*/
		return nil
	}

	if !strings.Contains(string(input), "404") {
		log.Println(string(input))
	}

	temp := make(map[string][]errorResponse)
	_ = json.Unmarshal(input, &temp)
	for key := range temp {
		for i := range temp[key] {
			_, exist = temp[key][i].Error["code"]
			if exist {
				return fmt.Errorf("%s - %s - %v", key, temp[key][i].Error["code"], temp[key][i])
			}
		}
	}

	return nil
}

func progressBar(completed, total int) {
	progress := float64(completed) / float64(total) * 100.0
	s := ("[")
	for pct := 0.0; pct <= 100.0; pct += 10.0 {
		if pct <= progress {
			s += "#"
		} else {
			s += "-"
		}
	}
	s += fmt.Sprintf("] %s%% completed", strconv.FormatFloat(progress, 'f', 2, 64))

	log.WithField("Progress", s).Info("Processing")
}
