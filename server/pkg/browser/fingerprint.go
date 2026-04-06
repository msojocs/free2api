package browser

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"
)

type Fingerprint struct {
	UserAgent string
	Platform  string
	Language  string
	Timezone  string
	ScreenRes string
	DeviceID  string
}

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
}

var platforms = []string{"Win32", "MacIntel", "Linux x86_64"}
var resolutions = []string{"1920x1080", "2560x1440", "1366x768", "1440x900"}
var timezones = []string{"America/New_York", "America/Los_Angeles", "Europe/London", "Asia/Shanghai"}

func randInt(max int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(n.Int64())
}

func Generate() *Fingerprint {
	deviceIDBytes := make([]byte, 16)
	if _, err := rand.Read(deviceIDBytes); err != nil {
		// Fall back to timestamp-based bytes on failure
		ts := uint64(time.Now().UnixNano())
		for i := 0; i < 8; i++ {
			deviceIDBytes[i] = byte(ts >> (i * 8))
		}
	}
	return &Fingerprint{
		UserAgent: userAgents[randInt(len(userAgents))],
		Platform:  platforms[randInt(len(platforms))],
		Language:  "en-US",
		Timezone:  timezones[randInt(len(timezones))],
		ScreenRes: resolutions[randInt(len(resolutions))],
		DeviceID: fmt.Sprintf("%s-%s-%s-%s-%s",
			hex.EncodeToString(deviceIDBytes[0:4]),
			hex.EncodeToString(deviceIDBytes[4:6]),
			hex.EncodeToString(deviceIDBytes[6:8]),
			hex.EncodeToString(deviceIDBytes[8:10]),
			hex.EncodeToString(deviceIDBytes[10:16])),
	}
}
