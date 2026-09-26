package job

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"time"

	"github.com/codebymagician/zenaclub-jobs/config"
	external_api "github.com/codebymagician/zenaclub-jobs/external-api"
	"github.com/codebymagician/zenaclub-jobs/utils"
)

// Ported from ringly-jobs/crm-jobs/job/subscription-due-charges.go verbatim
// (only the import paths changed) -- same backend endpoint contract, same
// reasoning throughout, so keep the two in sync if either changes.

// subscriptionDueChargesInterval only bounds how late a renewal can be, not
// whether it happens: the backend selects every mandate whose next_charge_at has
// already passed, so a missed tick is picked up by the next one. Every 5 minutes
// keeps a renewal at most 5 minutes late; ticks are wall-clock aligned (see
// nextAlignedTick) so that ceiling holds regardless of when this process last
// started or restarted.
const subscriptionDueChargesInterval = 5 * time.Minute

// subscriptionRunInFlight stops a slow run from being overlapped by the next
// tick. The backend keeps charging after our HTTP timeout fires, so without
// this a run lasting over an hour would have a second run charging the same
// mandates alongside it.
var subscriptionRunInFlight atomic.Bool

// StartSubscriptionDueChargesScheduler drives recurring subscription debits.
// Without it, mandates authorize and take their first charge but are never
// billed again -- next_charge_at passes and nothing reads it.
//
// Deliberately no run-on-startup call: this moves real money, so a crash-loop
// would retry charges as fast as the process restarts. The flip side is that
// a process restarting more often than the tick interval never charges at
// all, so the startup log below states the cadence -- its absence from the
// logs is the symptom to look for.
func StartSubscriptionDueChargesScheduler(
	ctx context.Context,
	cfg *config.Config,
	logger *utils.Logger,
) {
	// Checked at startup, not at the first tick: a missing secret otherwise
	// surfaces only as a delayed error line at the next tick, and looks
	// exactly like the silent no-billing outage this job exists to fix.
	if os.Getenv("CRON_JOB_SECRET") == "" {
		logger.Error("CRON_JOB_SECRET is not set — subscription renewals will NOT be charged")
	}

	logger.Info(fmt.Sprintf("Subscription due-charges scheduler started, wall-clock aligned every %s", subscriptionDueChargesInterval))

	go func() {
		for {
			wait := time.Until(nextAlignedTick(time.Now(), subscriptionDueChargesInterval))
			timer := time.NewTimer(wait)

			select {
			case <-timer.C:
				RunSubscriptionDueCharges(logger)

			case <-ctx.Done():
				timer.Stop()
				logger.Info("Subscription due-charges scheduler stopped")
				return
			}
		}
	}()
}

// nextAlignedTick returns the next wall-clock boundary that is a multiple of
// interval since the zero time (e.g. interval=5m -> :00, :05, :10, ...), strictly
// after now. Recomputed from the current time on every loop iteration rather
// than driven by a single long-lived time.Ticker, so alignment self-corrects
// instead of accumulating drift over a long-running process.
func nextAlignedTick(now time.Time, interval time.Duration) time.Time {
	return now.Truncate(interval).Add(interval)
}

func RunSubscriptionDueCharges(logger *utils.Logger) {
	if !subscriptionRunInFlight.CompareAndSwap(false, true) {
		logger.Error("Subscription due-charges run skipped: the previous run has not returned yet")
		return
	}
	defer subscriptionRunInFlight.Store(false)

	logger.Info("Subscription due-charges run started")

	status, body, err := external_api.RunSubscriptionDueCharges()

	if err != nil {
		// A timeout is not a failed run. Our client gave up; the backend is
		// still charging people. Logging it as a failure invites someone to
		// re-trigger the endpoint, which would charge alongside the run still
		// in flight -- so it gets its own message.
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			logger.Error("Subscription due-charges client timed out — the run is STILL EXECUTING server-side, outcome unknown. Do not re-trigger; check backend logs.")
			return
		}

		logger.Error(fmt.Sprintf("Subscription due-charges run failed (status %d): %s — %s", status, err, string(body)))
		return
	}

	// The body carries due/charged/skipped/failed. A 200 alone says nothing:
	// the backend logs and swallows every per-mandate error, so a run where
	// every charge failed answers 200 exactly like a clean one.
	logger.Info(fmt.Sprintf("Subscription due-charges run finished (status %d): %s", status, string(body)))
}
