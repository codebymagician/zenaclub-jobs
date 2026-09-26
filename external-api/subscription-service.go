package external_api

import (
	"fmt"
	"os"
	"time"

	"github.com/codebymagician/zenaclub-jobs/utils"
)

// runDueChargesTimeout is deliberately far above the APIClient default of 10s:
// the endpoint charges every due mandate synchronously in one request, so its
// duration grows with the number of subscribers. Timing out client-side would
// not stop the server mid-run -- it would just leave us blind to the outcome.
const runDueChargesTimeout = 5 * time.Minute

// zenaclubBackendURL defaults to the same-VM assumption crm-jobs makes about
// ringly-backend (this job and the backend run as sibling systemd services on
// zenaclub-backend-vm) -- overridable for local testing against a dev backend.
func zenaclubBackendURL() string {
	if url := os.Getenv("ZENACLUB_BACKEND_URL"); url != "" {
		return url
	}
	return "http://localhost:8080/"
}

// RunSubscriptionDueCharges triggers zenaclub-backend's subscription charge
// run. The endpoint is guarded by a shared secret rather than JWT auth, since
// it was built for an external scheduler; CRON_JOB_SECRET must match the
// value the backend process was started with.
func RunSubscriptionDueCharges() (int, []byte, error) {
	secret := os.Getenv("CRON_JOB_SECRET")
	if secret == "" {
		return 0, nil, fmt.Errorf("CRON_JOB_SECRET is not set")
	}

	apiClient := utils.NewAPIClient(zenaclubBackendURL())
	apiClient.Headers = map[string]string{"X-Cron-Secret": secret}
	apiClient.Client.Timeout = runDueChargesTimeout

	return apiClient.Post("subscription-service/v1/internal/run-due-charges", struct{}{})
}
