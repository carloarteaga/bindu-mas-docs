# Cron Job Strings in Bindu

Bindu uses a powerful scheduling engine based on the Go library `github.com/robfig/cron/v3`. It supports a **6-field cron format (with seconds)** as well as rich extensions like interval-based `@every` descriptors.

---

## 1. Standard 6-Field Cron Format

Unlike traditional UNIX cron (which uses 5 fields), Bindu requires exactly **6 fields** to support sub-minute scheduling accuracy.

```
 ┌──────────── second (0 - 59)
 │ ┌────────── minute (0 - 59)
 │ │ ┌──────── hour (0 - 23)
 │ │ │ ┌────── day of month (1 - 31)
 │ │ │ │ ┌──── month (1 - 12)
 │ │ │ │ │ ┌── day of week (0 - 6, where 0 is Sunday)
 │ │ │ │ │ │
 * * * * * *
```

### Common Expression Examples

* **Every minute (at second 0):**
  ```cron
  0 */1 * * * *
  ```
* **Every 5 minutes (at second 0):**
  ```cron
  0 */5 * * * *
  ```
* **Every hour (at minute 0, second 0):**
  ```cron
  0 0 * * * *
  ```
* **Every day at midnight (00:00:00):**
  ```cron
  0 0 0 * * *
  ```
* **Every day at 03:30 AM:**
  ```cron
  0 30 3 * * *
  ```

> [!IMPORTANT]
> A traditional 5-field cron expression (e.g. `*/5 * * * *`) will be rejected by the Bindu validator as invalid. Always specify the second field.

---

## 2. Standard Descriptor Shortcuts

You can use the following standard descriptors to replace the 6-field numeric expressions for daily/hourly tasks:

| Descriptor | Equivalent Numeric Expression | Description |
| :--- | :--- | :--- |
| `@hourly` | `0 0 * * * *` | Run once an hour, at the beginning of the hour. |
| `@daily` / `@midnight` | `0 0 0 * * *` | Run once a day, at midnight. |
| `@weekly` | `0 0 0 * * 0` | Run once a week, at midnight on Sunday morning. |
| `@monthly` | `0 0 0 1 * *` | Run once a month, at midnight on the first day of the month. |
| `@yearly` / `@annually` | `0 0 0 1 1 *` | Run once a year, at midnight on Jan 1st. |

---

## 3. The `@every <duration>` Extension (Highly Recommended)

Bindu supports the `@every <duration>` syntax, which schedules the task to run at fixed intervals relative to when the runner starts. 

This is particularly useful when you want tasks to run at a specific periodicity without worrying about hour or minute boundary alignment.

### Syntax

```cron
@every <number><unit>
```

Supported units are:
* `s` or `second` (seconds)
* `m` or `minute` (minutes)
* `h` or `hour` (hours)

### Examples

* **Every 10 seconds:** `@every 10s`
* **Every 45 minutes:** `@every 45m`
* **Every 1 hour and 30 minutes:** `@every 1h30m`

---

## 4. Key Differences: `*/N` vs `@every N`

When scheduling periodic tasks (like every 45 minutes), it is crucial to understand the difference between boundary-based scheduling and interval-based scheduling:

### A. Boundary-based: `0 */45 * * * *`
* **Behavior:** Triggers only when the minute of the hour is a multiple of 45 (minutes `:00` and `:45`).
* **Effect:** The task runs at `XX:00` and `XX:45`. This results in uneven intervals: **45 minutes** between `:00` and `:45`, and only **15 minutes** between `:45` and the next hour `:00`.
* **Risk:** If the task has a deduplication/safety rate limit set to 45 minutes, the trigger at `:00` (only 15 minutes after the `:45` trigger) will be discarded, resulting in an effective execution interval of **60 minutes**.

### B. Interval-based: `@every 45m`
* **Behavior:** Triggers exactly every 45 minutes from the start time, completely ignoring clock-hour boundaries.
* **Effect:** The task runs at `T+45m`, `T+90m`, `T+135m`, etc.
* **Benefit:** The interval between executions is always exactly **45 minutes**, making it much more predictable and preventing safety-limit conflicts.