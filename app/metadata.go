package app

import (
	"strconv"
	"time"
)

func GetMetadataJson(inputData Request) MetaData {
	var metadata MetaData
	metadata.ID = inputData.PostId
	metadata.Type = "image"
	// metadata.EsPath = append(metadata.EsPath, esPath)
	// metadata.Owner = inputData.
	// metadata.Caption = inputData.Caption
	now := time.Now()
	sec := now.Unix() // number of seconds since January 1, 1970 UTC
	metadata.PostedTime = strconv.FormatUint(uint64(sec), 10)
	metadata.Active = true
	return metadata
}
