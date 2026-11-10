package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/cli"
	"github.com/turboBasic/template-go-cli-app/internal/config"
	"github.com/turboBasic/template-go-cli-app/internal/infra/memory"
	"github.com/turboBasic/template-go-cli-app/internal/infra/onepassword"
	"github.com/turboBasic/template-go-cli-app/internal/infra/retry"
)

func newDemoResourceFactory(resources app.ResourceReader) cli.DemoFactory {
	return func(
		_ context.Context, _ config.Config,
		effective cli.DemoConfig,
		logger *slog.Logger,
		progressReporter app.ProgressReporter,
	) (*app.ResourceService, func(context.Context) error, error) {
		policy := effective.Retry.Demo.FlakyDependency
		flaky, err := retry.NewStatusReader(
			memory.NewFlakyReader(app.FlakyFailures),
			retry.Policy{
				MaxAttempts:  policy.MaxAttempts,
				InitialDelay: policy.InitialDelay,
				MaxDelay:     policy.MaxDelay,
				Jitter:       policy.Jitter,
			},
			retry.RealClock{},
			rand.New(rand.NewSource(time.Now().UnixNano())),
			retry.WithProgress(progressReporter),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("configure status retries: %w", err)
		}
		return app.NewResourceService(
			resources,
			logger,
			progressReporter,
			app.WithCredential(
				onepassword.NewResolver(onepassword.ExecRunner{}),
				effective.Demo.SecretCredential.Reference,
			),
			app.WithFlakyDependency(flaky),
		), nil, nil
	}
}
