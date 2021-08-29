package metadata

import (
	"encoding/json"
	"strconv"
	"time"
)

type InputData struct {
	ID      string
	User    string
	Data    string
	Caption string
	GeoId   string
}

func GetMetadataJson(reqBody []byte, esPath string) (MetaData, error) {
	var inputData InputData
	json.Unmarshal(reqBody, &inputData)
	err := ValidateInput(inputData)
	var metadata MetaData
	if err == nil {
		metadata = fillImageMetadata(inputData, esPath)
	}
	return metadata, err
}

func fillImageMetadata(inputData InputData, esPath string) MetaData {
	var metadata MetaData
	metadata.ID = inputData.ID
	metadata.Type = "image"
	metadata.EsPath = append(metadata.EsPath, esPath)
	metadata.Length = 0
	metadata.PiecesCount = 1
	metadata.Owner = inputData.User
	metadata.Caption = inputData.Caption
	now := time.Now()
	sec := now.Unix() // number of seconds since January 1, 1970 UTC
	metadata.PostedTime = strconv.FormatUint(uint64(sec), 10)
	metadata.Active = true
	return metadata
}
