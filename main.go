package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/codebymagician/zenaclub-jobs/config"
	"github.com/codebymagician/zenaclub-jobs/job"
	"github.com/codebymagician/zenaclub-jobs/utils"
)

// Same shutdown shape as ringly-jobs/crm-jobs/main.go -- a rootCtx cancelled
// on SIGTERM/interrupt that every Start*Scheduler is handed, so a future job
// ported from crm-jobs plugs into the same lifecycle unchanged.
func main() {
	cfg := config.LoadConfig()
	logger := utils.NewLogger(cfg.LogLevel)
	logger.Info("zenaclub-jobs started")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	rootCtx, cancelRoot := context.WithCancel(context.Background())
	defer cancelRoot()

	job.StartSubscriptionDueChargesScheduler(rootCtx, cfg, logger)

	<-stop
	logger.Info("Shutting down gracefully")
	cancelRoot()
	fmt.Println("Exiting...")
}
