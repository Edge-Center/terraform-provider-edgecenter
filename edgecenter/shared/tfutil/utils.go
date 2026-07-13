package tfutil

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
)

func MapStructureDecoder(strct interface{}, v *map[string]interface{}, config *mapstructure.DecoderConfig) error {
	config.Result = strct
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	}

	return decoder.Decode(*v)
}

func ImportStringParser(infoStr string) (projectID int, regionID int, id3 string, err error) { //nolint:nonamedreturns
	log.Printf("[DEBUG] Input id string: %s", infoStr)
	infoStrings := strings.Split(infoStr, ":")
	if len(infoStrings) != 3 {
		err = fmt.Errorf("failed import: wrong input id: %s", infoStr)
		return
	}

	id1, id2, id3 := infoStrings[0], infoStrings[1], infoStrings[2]

	projectID, err = strconv.Atoi(id1)
	if err != nil {
		return
	}
	regionID, err = strconv.Atoi(id2)
	if err != nil {
		return
	}

	return
}

func GetOptByName(fields map[string]interface{}, name string) (map[string]interface{}, bool) {
	if _, ok := fields[name]; !ok {
		return nil, false
	}

	container, ok := fields[name].([]interface{})
	if !ok {
		return nil, false
	}

	if len(container) == 0 {
		return nil, false
	}

	opt, ok := container[0].(map[string]interface{})
	if !ok {
		return nil, false
	}

	return opt, true
}

func ImportStringParserSimple(infoStr string) (id1 string, id2 string, err error) { //nolint:nonamedreturns
	log.Printf("[DEBUG] Input id string: %s", infoStr)
	infoStrings := strings.Split(infoStr, ":")
	if len(infoStrings) != 2 {
		err = fmt.Errorf("failed import: wrong input id: %s", infoStr)
		return
	}

	id1, id2 = infoStrings[0], infoStrings[1]

	return
}
