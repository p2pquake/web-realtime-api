package selector

import (
	"log"
	"reflect"
)

func NeedsBroadcast(m map[string]interface{}, codes []int64) bool {
	log.Printf("message arrival: %v\n", m)
	c, ok := m["code"]
	if !ok {
		return false
	}

	v := reflect.ValueOf(c).Int()
	for _, code := range codes {
		if v == code {
			return true
		}
	}

	log.Printf("code %v is not allowed\n", c)
	return false
}
