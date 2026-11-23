Bindu BPR Plugins — Lifecycle & Semantics Guide

Audience: Maintainers, contributors, advanced users
Purpose: Define the canonical plugin contract and lifecycle

1. Canonical Definition of a Plugin

A Bindu plugin is defined by:

Code (implements behavior)

Metadata (declares semantics)

There is exactly one semantic model.

2. Single Source of Truth
Metadata Files
File	Purpose
runtime.json	Scheduling & execution triggers
manifest.json	Message I/O, SQL, capabilities
Rules

Cron must be declared in runtime.json

Message semantics must be declared in manifest.json

Runtime does not infer capabilities

3. Go SDK (Canonical)

Import path: bindunet/sdk

Provides:

Plugin context

Runtime helpers

Common types

Rules:

SDK is imported as a module

SDK code is never copied into plugins

Python SDK (sdk/python) is separate and language-specific

4. Plugin Lifecycle (End-to-End)
1) Template Generation

Tool: bindu-studio plugin init (optionally with `--template`)

Writes:

runtime.json

manifest.json

Cron is written by default

No logs at this stage (files only).

2) Certify Packaging

Tool: bindu-studio plugin certify

Reads:

runtime.json

manifest.json

Validates and stages bundles

Cron logs:

certify.runtime.cron

certify.runtime.staged.cron

3) Import / Stage

Extracts bundle

Validates runtime metadata

Cron log:

lifecycle.plugin.runtime.cron

4) Activate / Promote

Validates executable

Registers plugin

Cron log:

lifecycle.plugin.activate.runtime

5) Runtime Load (Engine Start / Hotload)

Reads deployed metadata

Registers schedules

Cron log:

bpr.plugin.runtime.cron

5. Cron Debugging Checklist

If cron is missing in the Console, check logs in order:

certify.runtime.cron

certify.runtime.staged.cron

lifecycle.plugin.runtime.cron

lifecycle.plugin.activate.runtime

bpr.plugin.runtime.cron

The first missing log identifies the failure stage.

6. Scheduler Semantics

All plugins (internal + BPR) use the same scheduler

Scheduler reads runtime.json.triggers.cron.schedule

“Run Now” invokes OnSchedule

There is no BPR RPC for cron.

7. Guardrails

Scheduled plugins must not have empty schedules

Certify or activation emits warnings if schedules are missing

Generator prevents incoherent plugins by construction

8. Final Principle

Plugins are declared, then executed.
Metadata is the contract.
Runtime enforces, not invents.
