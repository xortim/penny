package integration

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// SignRequest adds valid X-Slack-Request-Timestamp and X-Slack-Signature
// headers to the given request, computed from the body using HMAC-SHA256.
// This matches what slack.NewSecretsVerifier expects.
func SignRequest(req *http.Request, body []byte, signingSecret string) {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	sigBaseString := fmt.Sprintf("v0:%s:%s", ts, string(body))

	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(sigBaseString))
	signature := fmt.Sprintf("v0=%x", mac.Sum(nil))

	req.Header.Set("X-Slack-Request-Timestamp", ts)
	req.Header.Set("X-Slack-Signature", signature)
	req.Header.Set("Content-Type", "application/json")
}
