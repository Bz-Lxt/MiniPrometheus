package block

type Meta struct {
	ULID  string `json:"ulid"`
	MinT  int64  `json:"minTime"`
	MaxT  int64  `json:"maxTime"`
	Stats struct {
		NumSeries  int `json:"numSeries"`
		NumSamples int `json:"numSamples"`
		NumChunks  int `json:"numChunks"`
	} `json:"stats"`
}
